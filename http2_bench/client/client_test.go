package main

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"golang.org/x/net/http2"
)

func BenchmarkClient(b *testing.B) {
	t := (http.DefaultTransport.(*http.Transport))
	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	if err := http2.ConfigureTransport(t); err != nil {
		log.Fatal(err)
	}

	b.Run("benchmark", func(b *testing.B) {
	        for i := 0; i < b.N; i++ {
			request(b)
	        }
	})
}

func request(b *testing.B) {
	req := make([]byte, 10)
	for i, _ := range req {
		req[i] = byte(i)
	}

	addr := "localhost:8080"

	var httpResp *http.Response
	var err error
	httpResp, err = http.Post("https://" + addr, "application/grpc", bytes.NewReader(req))
	if err != nil {
		b.Errorf("post err: %v", err)
	}
	resp, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		b.Errorf("err happened. %v", err)
	}
	if err = httpResp.Body.Close(); err != nil {
		b.Errorf("close err: %v", err)
	}
	if bytes.Compare(resp, req) != 0 {
		b.Errorf("bad response. want %v; got %v", req, resp)
	}
}
