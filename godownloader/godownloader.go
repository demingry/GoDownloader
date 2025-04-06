package main

import (
	godownloader "GoDownload"
	godownloader_torrent "GoDownload/bittorrent"
	godownloader_simple "GoDownload/simple"

	"os"

	webview_go "github.com/webview/webview_go"
)

// TODO: way to handle errs.
func CommonDispatch(urlinput string, downloadtype string) {
	if downloadtype == "normal" {
		godownloader_simple.CheckURLValidAndDispatch(urlinput)
	} else if downloadtype == "torrent" {
		godownloader_torrent.Dispatch(`debian-12.10.0-amd64-netinst.iso.torrent`)
	}
}

// Initial GUI Window.
func InitialGUIWindow() *godownloader.ErrorContext {
	w := webview_go.New(false)
	defer w.Destroy()
	w.SetTitle("GO Downloader")
	w.SetSize(900, 700, webview_go.HintNone)

	w.Bind("Dispatch", CommonDispatch)

	// w.Bind("ParseURLAndDispatch", godownloader_simple.CheckURLValidAndDispatch)
	w.Bind("RequestResult", godownloader.RequestResult)

	html, err := os.ReadFile(`./index.html`)
	if err != nil {
		return &godownloader.ErrorContext{Message: err.Error(), Statuscode: 105}
	}
	w.SetHtml(string(html))
	w.Run()

	return nil
}

func main() {

	InitialGUIWindow()
}
