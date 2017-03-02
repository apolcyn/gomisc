package main

import (
	"crypto/tls"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

var (
	useGrpc = flag.Bool("do_grpc", false, "do grpc protocol msg")
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

const (
	expectedMsgLen = 2
)

func ReadGrpcMsg(r io.Reader, header []byte, body []byte) error {
	if _, err := io.ReadFull(r, header); err != nil {
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return io.EOF
		}
		log.Fatalf("read header: %v", err)
	}
	pf := header[0]
	if pf != 0 {
		log.Fatalf("expect not compressed")
	}
	length := binary.BigEndian.Uint32(header[1:])
	if length != expectedMsgLen {
		log.Fatalf("expected msg length %v; got %v", expectedMsgLen, length)
	}
	if len(body) != expectedMsgLen {
		panic("invalid")
	}
	if _, err := io.ReadFull(r, body); err != nil {
		log.Fatalf("read body: %v", err)
	}
	return nil
}

func WriteGrpcMsg(w io.Writer, body []byte, out []byte) {
	if len(out) != 5+len(body) {
		panic("invalid")
	}
	out[0] = 0
	binary.BigEndian.PutUint32(out[1:], uint32(len(body)))
	if n := copy(out[5:], body); n != len(body) {
		panic("bad copy")
	}
	if n, err := w.Write(out); err != nil || n != len(out) {
		log.Fatalf("write msg: %v", err)
	}
	w.(http.Flusher).Flush()
}

func grpcPingPong(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/grpc")
	w.Header().Set("Content-Length", "-1")
	w.(http.Flusher).Flush()
	w.Header().Set("Trailer:grpc-status", "0")
	header := make([]byte, 5)
	body := make([]byte, expectedMsgLen)
	out := make([]byte, expectedMsgLen+5)
	for {
		if err := ReadGrpcMsg(req.Body, header, body); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		WriteGrpcMsg(w, body, out)
	}
	req.Body.Close()
}

func pingPong(w http.ResponseWriter, req *http.Request) {
	w.(http.Flusher).Flush()
	io.Copy(&flushWriter{w}, req.Body)
	req.Body.Close()
	log.Println("done\n")
}

func main() {
	flag.Parse()
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

	handler := http.HandlerFunc(pingPong)
	if *useGrpc {
		log.Println("starting grpc")
		handler = http.HandlerFunc(grpcPingPong)
	}

	srv := &http.Server{Addr: "localhost:8080", TLSConfig: &config, Handler: handler}
	http2.ConfigureServer(srv, nil)
	tlsListener := tls.NewListener(l, &config)
	log.Fatal(srv.Serve(tlsListener))
}
