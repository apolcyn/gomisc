package main

import (
	"context"
	"crypto/tls"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/benchmark/grpc_testing"
	"google.golang.org/grpc/credentials"

	"golang.org/x/net/http2"
)

var (
	useGrpc = flag.Bool("grpc", false, "grpc instead of http2 client")
)

func main() {
	flag.Parse()
	t := (http.DefaultTransport.(*http.Transport))
	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	if err := http2.ConfigureTransport(t); err != nil {
		log.Fatal(err)
	}

	done := make(chan int, 1)
	go func() {
		time.Sleep(time.Second * 1)
		done <- 0
	}()
	if *useGrpc {
		grpcPingPong(done)
	} else {
		http2PingPong(done)
	}
}

func http2PingPong(done chan int) {
	addr := "localhost:8080"

	var httpResp *http.Response
	var httpReq *http.Request
	var err error
	pr, pw := io.Pipe()
	log.Println("new http2 post")
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
	log.Println("http2 count: " + strconv.FormatInt(int64(count), 10))
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
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	log.Println("dial grpc")
	conn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(creds), grpc.WithCodec(&byteBufCodec{}))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("new client")
	c := grpc_testing.NewBenchmarkServiceClient(conn)
	client, err := c.StreamingCall(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	write := func(out []byte) {
		err := client.(grpc.ClientStream).SendMsg(&out)
		if err != nil {
			log.Fatal(err)
		}
	}
	complete := false
	count := 0
	expected := make([]byte, 2)
	for i, _ := range expected {
		expected[i] = byte(i)
	}
	in := make([]byte, len(expected))
	for !complete {
		select {
		case <-done:
			complete = true
			break
		default:
			write(expected)
			if err := client.(grpc.ClientStream).RecvMsg(&in); err != nil {
				log.Fatal(err)
			}
			if len(in) != len(expected) {
				log.Fatalf("bad length")
			}
			count++
		}
	}
	if err := client.(grpc.ClientStream).CloseSend(); err != nil {
		log.Fatal(err)
	}
	log.Println("about to wait for EOF")
	if err := client.(grpc.ClientStream).RecvMsg(nil); err != io.EOF {
		log.Fatalf("expected EOF; got %v", err)
	}
	log.Println("about to wait for EOF")
	if err := conn.Close(); err != nil {
		log.Fatal(err)
	}
	log.Println("grpc count: " + strconv.FormatInt(int64(count), 10))
}
