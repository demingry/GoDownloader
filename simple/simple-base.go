package godownloader

import (
	godownloader "GoDownload"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type HTTPHeader struct {
	url    string
	header map[string][]string
}

type ErrorContext struct {
	Message    string
	Statuscode int
}

type FilePartion struct {
	partionID     int
	startposition int
	endposotion   int
	content       []byte
}

type FileDownloadTask struct {
	url        string
	filename   string
	savedpath  string
	filelength int
	md5        string
	partions   []FilePartion
}

// A HEAD request to fetch HTTP headers.
// Return a HTTPHeader or ErrorContext for error.
func FetchHTTPHeader(url string) (*HTTPHeader, *ErrorContext) {
	client := http.Client{}
	response, err := client.Head(url)
	if err != nil {
		return nil, &ErrorContext{Message: err.Error(), Statuscode: 100}
	}
	defer response.Body.Close()

	if len(response.Header) == 0 {
		return nil, &ErrorContext{Message: "[!]Cannot fetch HTTP header", Statuscode: 100}
	}
	return &HTTPHeader{url: url, header: response.Header}, nil
}

// Is server allowed range download by checking HTTP header field `Accepted-Ranges`
func IsServerAcceptedRange(httpheader *HTTPHeader) bool {
	if value, key := httpheader.header[`Accept-Ranges`]; key {
		switch strings.ToLower(fmt.Sprintf("%v", value)) {
		case `[bytes]`:
			return true
		case `[none]`:
			return false
		}
	}
	return false
}

// Initial FiledownloadTask struct and return.
func InitDownloadTask(httpheader *HTTPHeader) (*FileDownloadTask, *ErrorContext) {
	contentlength, err := strconv.Atoi(httpheader.header["Content-Length"][0])
	if err != nil {
		return nil, &ErrorContext{Message: err.Error(), Statuscode: 101}
	}

	returndownloadtask := new(FileDownloadTask)
	returndownloadtask.filelength = contentlength
	returndownloadtask.url = httpheader.url

	//Fixed numberofpart for test.
	//if mudule is not equals to zero, then the last part's positon points to remaining length.
	numberofpart := 5
	sizemudule := contentlength % numberofpart
	eachsizeofpart := contentlength / numberofpart
	if sizemudule != 0 {
		eachsizeofpart = contentlength/numberofpart - 1
	}

	returndownloadtask.partions = make([]FilePartion, numberofpart)

	for i := range numberofpart {
		if i == numberofpart-1 && sizemudule != 0 {
			returndownloadtask.partions[i] = FilePartion{partionID: i, startposition: i * eachsizeofpart, endposotion: contentlength - 1}
			break
		}
		returndownloadtask.partions[i] = FilePartion{partionID: i, startposition: i * eachsizeofpart, endposotion: (i+1)*eachsizeofpart - 1}
	}

	return returndownloadtask, nil
}

// Start goroutines to dowload.
func (task *FileDownloadTask) TaskRun() *ErrorContext {

	var wg sync.WaitGroup
	for t := range len(task.partions) {
		wg.Add(1)
		go func(partion *FilePartion) {
			defer wg.Done()
			request, _ := http.NewRequest(`GET`, task.url, nil)
			request.Header.Set(`Range`, fmt.Sprintf("bytes=%v-%v", partion.startposition, partion.endposotion))
			client := http.Client{}
			response, err := client.Do(request)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			defer response.Body.Close()

			content, err := io.ReadAll(response.Body)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			partion.content = content
			godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte(fmt.Sprintf("[!]File partion index: %v, %v-%v, received: %v\n",
				partion.partionID, partion.startposition, partion.endposotion, len(content))))
		}(&task.partions[t])
	}
	wg.Wait()

	return nil
}

// Merge all downloaded partion and compare to the file size.
func (task *FileDownloadTask) MergePartion() *ErrorContext {
	// file, err := os.Create(filepath.Join(task.savedpath, task.filename))
	file, err := os.Create(filepath.Join(`./`, `testfile.save`))
	if err != nil {
		return &ErrorContext{Message: err.Error(), Statuscode: 103}
	}
	defer file.Close()

	//MD5 checksum
	md5hash := md5.New()
	var receivedfilesize int
	for _, p := range task.partions {
		file.Write(p.content)
		md5hash.Write(p.content)
		receivedfilesize += len(p.content)
	}

	if receivedfilesize != task.filelength {
		return &ErrorContext{Message: fmt.Sprintf("[!]File partion missed.%v:%v", task.filelength, receivedfilesize)}
	}

	task.md5 = hex.EncodeToString(md5hash.Sum(nil))

	return nil
}
