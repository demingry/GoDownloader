package main

import (
	"net/url"
	"os"

	webview_go "github.com/webview/webview_go"
)

// TODO: IO buffer io.reader and io.writer or Channel
var messageBuffer [][]byte

func CheckURLValidAndDispatch(urlinput string) {
	go func(urlinput string) {
		messageBuffer = append(messageBuffer, []byte("[!]Task starting..."))
		urlparsed, e := url.ParseRequestURI(urlinput)
		if e != nil {
			messageBuffer = append(messageBuffer, []byte("[!URL is illigal"))
			return
		}
		if urlparsed.Scheme == "" && urlparsed.Host == "" {
			messageBuffer = append(messageBuffer, []byte("[!]URL is illigal"))
			return
		}

		httpheader, err := FetchHTTPHeader(urlinput)
		if err != nil {
			messageBuffer = append(messageBuffer, []byte(err.message))
			return
		}

		is_server_accepted_range := IsServerAcceptedRange(httpheader)
		if !is_server_accepted_range {
			messageBuffer = append(messageBuffer, []byte("[!]Range Download is not allowed for your url provided"))
			return
		}

		task, err := InitDownloadTask(httpheader)
		if err != nil {
			messageBuffer = append(messageBuffer, []byte(err.message))
			return
		}

		if err = task.TaskRun(); err != nil {
			messageBuffer = append(messageBuffer, []byte(err.message))
			return
		}

		if err = task.MergePartion(); err != nil {
			messageBuffer = append(messageBuffer, []byte(err.message))
			return
		}

		messageBuffer = append(messageBuffer, []byte("[!]Task finished, 0 error occured"))
		messageBuffer = append(messageBuffer, []byte("[!]MD5: "+task.md5))
	}(urlinput)
}

// Bind to frontpage and return 1 message each time.
func RequestResult() string {
	if len(messageBuffer) > 0 {
		queuehead := messageBuffer[0]
		messageBuffer = messageBuffer[1:]
		return string(queuehead)
	}

	return ""
}

// Initial GUI Window.
func InitialGUIWindow() *ErrorContext {
	w := webview_go.New(false)
	defer w.Destroy()
	w.SetTitle("GO Downloader")
	w.SetSize(900, 600, webview_go.HintNone)

	w.Bind("ParseURLAndDispatch", CheckURLValidAndDispatch)
	w.Bind("RequestResult", RequestResult)

	html, err := os.ReadFile(`./index.html`)
	if err != nil {
		return &ErrorContext{message: err.Error(), statuscode: 105}
	}
	w.SetHtml(string(html))
	w.Run()

	return nil
}
