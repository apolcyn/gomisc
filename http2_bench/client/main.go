package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
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

	reqBuffer := newRecvBuffer()
	respBuffer := newRecvBuffer()
	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * 2)
		doneChan <- true
	}()
	totalChan := make(chan int64, 1)
	go func() {
		count := 0
		defer reqBuffer.put(make([]byte, 0))
		defer func() { totalChan <- int64(count) }()
		for {
			select {
			case <-doneChan:
				return
			default:
			}
			requestResponse(reqBuffer, respBuffer)
			count += 1
		}
	}()
	log.Println("about to start up call")
	initCall(reqBuffer, respBuffer)
	total := <-totalChan
	log.Println("total: " + strconv.FormatInt(total, 10))
}

type recvBuffer struct {
	c       chan []byte
	mu      sync.Mutex
	backlog [][]byte
}

func newRecvBuffer() *recvBuffer {
	return &recvBuffer{
		c:       make(chan []byte, 1),
		backlog: make([][]byte, 0),
	}
}

func (r *recvBuffer) get() chan []byte {
	return r.c
}

func (r *recvBuffer) load() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.backlog) > 0 {
		select {
		case r.c <- r.backlog[0]:
			r.backlog = r.backlog[1:]
		default:
			return
		}
	}
}

func (r *recvBuffer) put(b []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.backlog) == 0 {
		select {
		case r.c <- b:
			return
		default:
		}
	}
	r.backlog = append(r.backlog, b)
}

func (r *recvBuffer) Write(b []byte) (int, error) {
	if len(b) != 10 {
		panic("wrong")
	}
	r.put(b)
	return 10, nil
}

func (r *recvBuffer) Read(dest []byte) (int, error) {
	b := <-r.get()
	r.load()

	if len(b) == 0 {
		log.Println("recvBuffer EOF reached")
		return 0, io.EOF
	}
	if len(b) != 10 {
		panic("something wrong")
	}

	n := copy(dest, b)
	if len(dest) < 10 {
		panic("wrong")
	}
	if n != 10 {
		panic("bad")
	}
	return len(b), nil
}

func initCall(reqBuffer *recvBuffer, respBuffer *recvBuffer) {
	addr := "localhost:8080"

	var httpResp *http.Response
	var httpReq *http.Request
	var err error
	httpReq, err = http.NewRequest("POST", "https://"+addr, reqBuffer)
	httpReq.ContentLength = -1
	httpReq.Proto = "HTTP/2"
	if err != nil {
		log.Fatal(err)
	}
	httpResp, err = http.DefaultClient.Do(httpReq)
	defer httpResp.Body.Close()
	for {
		resp := make([]byte, 10)
		_, err = io.ReadFull(httpResp.Body, resp)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			panic("")
		}
		if err != nil {
			break
		}
		_, err = io.Copy(respBuffer, bytes.NewReader(resp))
	}
}

func requestResponse(reqBuffer *recvBuffer, respBuffer *recvBuffer) {
	req := make([]byte, 10)
	for i, _ := range req {
		req[i] = byte(i)
	}
	reqBuffer.put(req)
	resp := make([]byte, 10)
	_, err := io.ReadFull(respBuffer, resp)
	if err != nil {
		log.Fatalf("err happened. %v", err)
	}
	if bytes.Compare(resp, req) != 0 {
		log.Fatalf("bad response. want %v; got %v", req, resp)
	}
}
