package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/gobench", func(w http.ResponseWriter, req *http.Request) {
		b := make([]byte, 10)
		for {
			_, err := io.ReadFull(req.Body, b)
			if err == io.EOF {
				break
			} else if err != nil {
				panic("read error")
			}
			if _, err = w.Write(b); err != nil {
				panic("something happened")
			}
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
