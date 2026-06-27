package base

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

func ParseProxy(proxy string) (server string) {
	if iport, err := strconv.Atoi(proxy); err == nil {
		if iport > 0 && iport <= 65535 {
			return fmt.Sprintf("127.0.0.1:%d", iport)
		}
		return ""
	}

	return proxy
}

func FindUserPath(exeDir, exeName string, portable bool) string {
	var dataPath string
	for _, dir := range []string{exeName, "." + exeName, exeName + ".data", "data"} {
		dir := filepath.Join(exeDir, dir)
		if stat, _ := os.Stat(dir); stat != nil && stat.IsDir() {
			dataPath = dir
			break
		}
	}
	if dataPath == "" {
		if portable {
			dataPath = filepath.Join(exeDir, exeName+".data")
		} else {
			dir, _ := os.UserConfigDir()
			dataPath = filepath.Join(dir, exeName)
		}
	}

	return dataPath
}

func FindServeUrl(ctx context.Context, appUrl string, backend string) (string, error) {
	if strings.HasPrefix(appUrl, "file://") {
		if port, err := ServeFS(ctx, appUrl[7:], backend); err == nil {
			appUrl = fmt.Sprintf("http://127.0.0.1:%d", port)
		} else {
			return "", nil
		}
	}
	return appUrl, nil
}

func ServeFS(ctx context.Context, dir string, backend string) (int, error) {
	started := make(chan int, 1)
	errc := make(chan error, 1)
	defer close(started)

	mux := http.NewServeMux()
	mux.Handle("GET /*", http.FileServerFS(os.DirFS(dir)))

	//backend: route=/api,url=http://127.0.0.1:7993,strip=/api
	if backend != "" {
		var route, target, strip string
		for item := range strings.SplitSeq(backend, ",") {
			k, v, ok := strings.Cut(item, "=")
			if !ok {
				target = cmp.Or(target, item)
			} else {
				switch strings.ToLower(k) {
				case "route":
					route = cmp.Or(route, v)
				case "url":
					target = cmp.Or(target, v)
				case "strip":
					strip = cmp.Or(strip, v)
				}
			}
		}

		if target != "" {
			if targetUri, _ := url.Parse(route); targetUri != nil {
				hRoute := &httputil.ReverseProxy{
					Rewrite: func(r *httputil.ProxyRequest) {
						r.SetURL(targetUri)
						if targetUri.Scheme == "https" {
							r.Out.Host = targetUri.Host
						}
						if strip != "" {
							r.Out.URL.Path = strings.TrimPrefix(r.Out.URL.Path, strip)
						}
						r.SetXForwarded()
						slog.Debug("rewrite:", "out", r.Out.URL.String(), "in", r.In.URL.String())
					},
				}
				if route == "" {
					route = "/"
				}
				mux.Handle(route, hRoute)
			}
		}
	}

	s := &http.Server{Addr: "127.0.0.1:0", Handler: mux}
	go func() {
		s.BaseContext = func(l net.Listener) context.Context {
			started <- l.Addr().(*net.TCPAddr).Port
			return ctx
		}
		errc <- s.ListenAndServe()
	}()

	select {
	case port := <-started:
		return port, nil
	case err := <-errc:
		return 0, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func MessageBoxW(msg, title string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")

	uMsg, _ := syscall.UTF16PtrFromString(msg)
	uTitle, _ := syscall.UTF16PtrFromString(title)

	// 调用 MessageBoxW 显示弹窗[reference:20][reference:21]
	messageBoxW.Call(0, uintptr(unsafe.Pointer(uMsg)), uintptr(unsafe.Pointer(uTitle)), 0 /*按钮类型，0 表示只有“确定”按钮*/) //nolint:errcheck
}

func AppendVebViewAdditionalBrowserArguments(format string, args ...any) {
	const key = "WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS"
	os.Setenv(key, strings.TrimSpace(os.Getenv(key)+" "+fmt.Sprintf(format, args...)))
}
