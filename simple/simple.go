package godownloader

import (
	godownloader "GoDownload"
	"net/url"
)

func CheckURLValidAndDispatch(urlinput string) {
	go func(urlinput string) {
		godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte("[!]Task starting..."))
		urlparsed, e := url.ParseRequestURI(urlinput)
		if e != nil {
			godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte("[!URL is illigal"))
			return
		}
		if urlparsed.Scheme == "" && urlparsed.Host == "" {
			godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte("[!]URL is illigal"))
			return
		}

		httpheader, err := FetchHTTPHeader(urlinput)
		if err != nil {
			godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte(err.Message))
			return
		}

		is_server_accepted_range := IsServerAcceptedRange(httpheader)
		if !is_server_accepted_range {
			godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte("[!]Range Download is not allowed for your url provided"))
			return
		}

		task, err := InitDownloadTask(httpheader)
		if err != nil {
			godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte(err.Message))
			return
		}

		if err = task.TaskRun(); err != nil {
			godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte(err.Message))
			return
		}

		if err = task.MergePartion(); err != nil {
			godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte(err.Message))
			return
		}

		godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte("[!]Task finished, 0 error occured"))
		godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte("[!]MD5: "+task.md5))
	}(urlinput)
}
