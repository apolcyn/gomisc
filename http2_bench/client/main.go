package main

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc/benchmark/grpc_testing"

	"golang.org/x/net/http2"
	"google.golang.org/grpc"
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
	//grpcPingPong(done)
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

type byteBufCodec struct {
}

func (*byteBufCodec) Marshal(v interface{}) ([]byte, error) {
	return *v.(*[]byte), nil
}

func (*byteBufCodec) Unmarshal(source []byte, v interface{}) error {
	copy(*v.(*[]byte), source)
	return nil
}

func (*byteBufCodec) String() string {
	return "bytebufcodec"
}

func grpcPingPong(done chan int) {
	conn, err := grpc.Dial("localhost:8081", grpc.WithInsecure(), grpc.WithCodec(&byteBufCodec{}))
	if err != nil {
		log.Fatal(err)
	}
	c := grpc_testing.NewBenchmarkServiceClient(conn)
	client, err := c.StreamingCall(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	read := func(dest []byte) bool {
		err := client.(grpc.ClientStream).RecvMsg(&dest)
		if err != nil {
			if err == io.EOF {
				return true
			}
			log.Fatal(err)
		}
		return false
	}
	write := func(out []byte) {
		err := client.(grpc.ClientStream).SendMsg(&out)
		if err != nil {
			log.Fatal(err)
		}
	}
	complete := false
	count := 0
	expected := make([]byte, 10)
	for i, _ := range expected {
		expected[i] = byte(i)
	}
	for !complete {
		select {
		case <-done:
			complete = true
			break
		default:
			write(expected)
			in := make([]byte, len(expected))
			read(in)
			count++
		}
	}
	log.Println("count: " + strconv.FormatInt(int64(count), 10))
}
