package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
)

func pingPong(w http.ResponseWriter, req *http.Request) {
	expect := make([]byte, 10)
	for i, _ := range expect {
		expect[i] = byte(i)
	}
	b := make([]byte, 10)
	for {
		_, err := io.ReadFull(req.Body, b)
		if bytes.Compare(expect, b) != 0 {
			log.Println("want %v; got %v", expect, b)
		}
		if err == io.EOF {
			req.Body.Close()
			break
		} else if err != nil {
			panic("read error")
		}
		if _, err = w.Write(b); err != nil {
			panic("something happened")
		}
		w.(http.Flusher).Flush()
	}
}

func main() {
	l, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}
	var config tls.Config
	var certFile = "server.crt"
	var keyFile = "server.key"
	config.NextProtos = []string{"h2"}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)

	srv := &http.Server{Addr: "localhost:8080", TLSConfig: &config, Handler: http.HandlerFunc(pingPong)}
	tlsListener := tls.NewListener(l, &config)
	log.Fatal(srv.Serve(tlsListener))
}
