[![Go](https://github.com/cnk3x/win-webapp-gen/webview2/actions/workflows/go.yml/badge.svg)](https://github.com/cnk3x/win-webapp-gen/webview2/actions/workflows/go.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/cnk3x/win-webapp-gen/webview2)](https://goreportcard.com/report/github.com/cnk3x/win-webapp-gen/webview2) [![Go Reference](https://pkg.go.dev/badge/github.com/cnk3x/win-webapp-gen/webview2.svg)](https://pkg.go.dev/github.com/cnk3x/win-webapp-gen/webview2)

# go-webview2

This package provides an interface for using the Microsoft Edge WebView2 component with Go. It is based on [webview/webview](https://github.com/webview/webview) and provides a compatible API.

Please note that this package only supports Windows, since it provides functionality specific to WebView2. If you wish to use this library for Windows, but use webview/webview for all other operating systems, you could use the [go-webview-selector](https://github.com/jchv/go-webview-selector) package instead. However, you will not be able to use WebView2-specific functionality.

If you wish to build desktop applications in Go using web technologies, please consider [Wails](https://wails.io/). It uses go-webview2 internally on Windows.

## Demo

If you are using Windows 10+, the WebView2 runtime should already be installed. If you don't have it installed, you can download and install a copy from Microsoft's website:

[WebView2 runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)
