//go:build !creator

package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/jchv/go-webview2"
)

func main() {
	err := RunApp(context.Background())
	if err != nil {
		MessageBoxW(err.Error(), "错误")
	}
}

// RunApp 以服务模式运行
func RunApp(ctx context.Context) error {
	var app WebApp

	data, _, _, err := readExeExtra(exePath)
	if err == nil {
		if len(data) > 0 {
			err = json.Unmarshal(data, &app)
		} else {
			err = fmt.Errorf("应用被破坏")
		}
	}

	if app.Proxy != "" {
		u, _ := url.Parse(app.Proxy)
		if u.Scheme == "" {
			u.Scheme = "http"
		}

		proxyUrl := u.String()

		os.Setenv("http_proxy", proxyUrl)
		os.Setenv("https_proxy", proxyUrl)
		os.Setenv("all_proxy", proxyUrl)

		os.Setenv("HTTP_PROXY", proxyUrl)
		os.Setenv("HTTPS_PROXY", proxyUrl)
		os.Setenv("ALL_PROXY", proxyUrl)

		os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--proxy-server="+u.Host)
	}

	dataPath := app.DataPath
	if dataPath != "" {
		dataPath, _ = filepath.Abs(dataPath)
	} else {
		dirs := []string{
			exeName,
			exeName + ".data",
			exeName + "." + exeName,
			"data",
		}
		for _, dir := range dirs {
			if stat, _ := os.Stat(dir); stat != nil && stat.IsDir() {
				dataPath, _ = filepath.Abs(dir)
				break
			}
		}
		if dataPath == "" {
			dir, _ := os.UserConfigDir()
			dataPath = filepath.Join(dir, exeName)
		}
	}

	appUrl := app.Url

	switch {
	case appUrl == "":
		err = fmt.Errorf("没有指定目标地址")
	case strings.HasPrefix(appUrl, "file://"):
		assetDir := app.Url[7:]
		var port int
		port, err = ServeFS(ctx, assetDir)
		if err == nil {
			appUrl = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
	}

	wv := webview2.NewWithOptions(webview2.WebViewOptions{Debug: app.Debug, DataPath: dataPath, WindowOptions: webview2.WindowOptions{Title: app.Name, IconId: 1}})
	if wv == nil {
		return fmt.Errorf("Failed to load webview.")
	}
	defer wv.Destroy()

	wv.SetSize(cmp.Or(app.Width, 1280), cmp.Or(app.Height, 600), webview2.HintFixed)

	if err != nil {
		wv.SetHtml(err.Error())
	} else {
		wv.Navigate(appUrl)
	}
	wv.Run()
	return nil
}

func ServeFS(ctx context.Context, dir string) (int, error) {
	started := make(chan int, 1)
	errc := make(chan error, 1)
	defer close(started)

	s := &http.Server{Addr: "127.0.0.1:0", Handler: http.FileServerFS(os.DirFS(dir))}
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
