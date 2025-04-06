package godownloader

// TODO: IO buffer io.reader and io.writer or Channel
var MessageBuffer [][]byte

type ErrorContext struct {
	Message    string
	Statuscode int
}

// Bind to frontpage and return 1 Message each time.
func RequestResult() string {
	if len(MessageBuffer) > 0 {
		queuehead := MessageBuffer[0]
		MessageBuffer = MessageBuffer[1:]
		return string(queuehead)
	}

	return ""
}

func CommonDispatch(urlinput string, downloadtype string) {
	if downloadtype == "normal" {
	} else if downloadtype == "torrent" {
	}
}
