package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

type flushWriter struct {
	hw http.ResponseWriter
}

func (w *flushWriter) Write(p []byte) (int, error) {
	var n int
	var err error
	if n, err = w.hw.Write(p); err != nil {
		log.Fatal(err)
	}
	w.hw.(http.Flusher).Flush()
	return n, nil
}

func pingPong(w http.ResponseWriter, req *http.Request) {
	w.(http.Flusher).Flush()
	io.Copy(&flushWriter{w}, req.Body)
	req.Body.Close()
	log.Println("done\n")
}

func main() {
	l, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}
	var config tls.Config
	certFile := "server.crt"
	keyFile := "server.key"
	config.NextProtos = []string{"h2"}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{Addr: "localhost:8080", TLSConfig: &config, Handler: http.HandlerFunc(pingPong)}
	http2.ConfigureServer(srv, nil)
	tlsListener := tls.NewListener(l, &config)
	log.Fatal(srv.Serve(tlsListener))
}
