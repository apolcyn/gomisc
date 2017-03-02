#!/bin/bash
protoc -I ./testing ./testing/*.proto --go_out=plugins=grpc:testing
