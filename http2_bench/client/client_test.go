package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"sync"
	"testing"

	"golang.org/x/net/http2"
)

func BenchmarkClient(b *testing.B) {
	t := (http.DefaultTransport.(*http.Transport))
	if err := http2.ConfigureTransport(t); err != nil {
		log.Fatal(err)
	}

	b.Run("benchmark", func(b *testing.B) {
		reqBuffer := newRecvBuffer()
		respBuffer := newRecvBuffer()
		respBufferReader := &recvBufferReader{r:respBuffer}
		go func() {
			defer reqBuffer.put(make([]byte, 0))
		        for i := 0; i < b.N; i++ {
				requestResponse(b, reqBuffer, respBufferReader)
		        }
		}()
		initCall(b, reqBuffer, respBuffer)
	})
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
	select {
	case r.c <- b:
		return
	default:
		r.backlog = append(r.backlog, b)
	}
}

func (r *recvBuffer) Write(b []byte) (int, error) {
	if len(b) != 10 {
		panic("wrong")
	}
	r.put(b)
	return 10, nil
}

type recvBufferReader struct {
	r *recvBuffer
}

func (r *recvBufferReader) Read(dest []byte) (int, error) {
	b := <-r.r.get()
	r.r.load()

	if len(b) == 0 {
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

func initCall(b *testing.B, reqBuffer *recvBuffer, respBuffer *recvBuffer) {
	addr := "localhost:8080"

	var httpResp *http.Response
	var httpReq *http.Request
	var err error
	httpReq, err = http.NewRequest("POST", "http://"+addr, &recvBufferReader{r:reqBuffer})
	if err != nil {
		log.Fatal(err)
	}
	httpResp, err = http.DefaultClient.Do(httpReq)
	defer httpResp.Body.Close()
	panic("")
	_, err = io.Copy(respBuffer, httpResp.Body)
	if err != nil {
		b.Errorf("post err: %v", err)
	}
}

func requestResponse(b *testing.B, reqBuffer *recvBuffer, respBufferReader *recvBufferReader) {
	req := make([]byte, 10)
	for i, _ := range req {
		req[i] = byte(i)
	}
	reqBuffer.put(req)
	resp := make([]byte, 10)
	_, err := io.ReadFull(respBufferReader, resp)
	if err != nil {
		b.Errorf("err happened. %v", err)
	}
	if bytes.Compare(resp, req) != 0 {
		b.Errorf("bad response. want %v; got %v", req, resp)
	}
}
