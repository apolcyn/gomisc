package main

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/http2"
)

func main() {
	t := (http.DefaultTransport.(*http.Transport))
	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	if err := http2.ConfigureTransport(t); err != nil {
		log.Fatal(err)
	}

	log.Println("about to start up call")
	done := make(chan int, 1)
	go func() {
		time.Sleep(time.Second * 10)
		done <- 0
	}()
	http2PingPong(done)
}

func http2PingPong(done chan int) {
	addr := "localhost:8080"

	var httpResp *http.Response
	var httpReq *http.Request
	var err error
	pr, pw := io.Pipe()
	httpReq, err = http.NewRequest("POST", "https://"+addr, ioutil.NopCloser(pr))
	httpReq.ContentLength = -1
	httpReq.Proto = "HTTP/2"
	if err != nil {
		log.Fatal(err)
	}
	httpResp, err = http.DefaultClient.Do(httpReq)
	write := func(req []byte) {
		if _, err := pw.Write(req); err != nil {
			log.Fatal(err)
		}
	}
	read := func() bool {
		resp := make([]byte, 10)
		_, err = io.ReadFull(httpResp.Body, resp)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			panic("")
		}
		if err != nil {
			return true
		}
		return false
	}
	req := make([]byte, 10)
	for i, _ := range req {
		req[i] = byte(i)
	}
	complete := false
	count := 0
	for !complete {
		select {
		case <-done:
			complete = true
			break
		default:
			write(req)
			read()
			count++
		}
	}
	pw.Close()
	if !read() {
		panic("")
	}
	log.Println("count: " + strconv.FormatInt(int64(count), 10))
	defer httpResp.Body.Close()
}
