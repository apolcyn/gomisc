package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

func pingPong(w http.ResponseWriter, req *http.Request) {
	expect := make([]byte, 10)
	for i, _ := range expect {
		expect[i] = byte(i)
	}
	b := make([]byte, 10)
	for {
	        _, err := io.ReadFull(req.Body, b)
	        if err != nil {
			break
	        }
	        if bytes.Compare(expect, b) != 0 {
			log.Println("want %v; got %v", expect, b)
	        }
	        if _, err = w.Write(b); err != nil {
			panic("something happened")
	        }
		w.(http.Flusher).Flush()
	}
	req.Body.Close()
}

func main() {
	l, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{Addr: "localhost:8080", Handler: http.HandlerFunc(pingPong)}
	http2.ConfigureServer(srv, nil)
	log.Fatal(srv.Serve(l))
}
