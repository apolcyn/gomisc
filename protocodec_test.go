package gomisc

import (
	"testing"

	"google.golang.org/grpc"
)

func BenchmarkProtoCodec(t *testing.B) {
	payload := &Payload{}
	req := &SimpleRequest{Payload: payload}
	codec := grpc.NewProtoCodec()
	for i := 0; i < t.N; i++ {
		x, err := codec.Marshal(req)
		if err != nil {
			panic("marshal err")
		}
		out := &SimpleRequest{}
		codec.Unmarshal(x, out)
		if len(out.Payload.Body) != len(payload.Body) {
			panic("failed")
		}
	}
}
