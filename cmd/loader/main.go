package main

import (
	"cmp"
	"context"
	"fmt"

	"github.com/cnk3x/win-webapp-gen/base"
	"github.com/cnk3x/win-webapp-gen/webview2"
	// "github.com/energye/wv"
)

func main() {
	if err := RunApp(context.Background()); err != nil {
		base.MessageBoxW(err.Error(), "错误")
	}
}

// RunApp 以服务模式运行
func RunApp(ctx context.Context) (err error) {
	var exePath, exeDir, exeName string
	if exePath, exeDir, exeName, _, err = base.GetExecutePath(); err != nil {
		return
	}

	var app base.WebApp
	if _, _, err = base.ReadExeExtra(exePath, &app); err != nil {
		return
	}

	if proxyServer := base.ParseProxy(app.ProxyServer); proxyServer != "" {
		base.AppendVebViewAdditionalBrowserArguments("--proxy-server=%s", proxyServer)
	}

	if app.Insecure {
		base.AppendVebViewAdditionalBrowserArguments("--disable-web-security --allow-insecure-content --ignore-certificate-errors")
	}

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:    app.Debug,
		DataPath: base.FindUserPath(exeDir, exeName, app.Portable),
		WindowOptions: webview2.WindowOptions{
			Title:  app.Name,
			IconId: 1,
			Width:  uint(cmp.Or(app.Width, 1280)),
			Height: uint(cmp.Or(app.Height, 600)),
		},
	})
	if w == nil {
		err = fmt.Errorf("Failed to load webview.")
		return
	}
	defer w.Destroy()

	if appUrl, err := base.FindServeUrl(ctx, app.Url, app.Backend); err != nil {
		w.SetHtml(err.Error())
	} else {
		w.Navigate(appUrl)
	}

	w.Run()
	return nil
}
