#!/usr/bin/env bash

# go generate

# generate stubs
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --go_out=plugins=grpc:. \
  api/hello.proto

# clean up generated stub for Gorm purposes.  Suppresses XXX (internal) fields.
protoc-go-inject-tag -XXX_skip=gorm \
  -input=./api/hello.pb.go

# generate reverse proxy
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --grpc-gateway_out=logtostderr=true:. \
  api/hello.proto

# generate swagger
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --swagger_out=logtostderr=true:swaggerui \
  api/hello.proto

# embed swaggerui in code that will be compiled into binary
statik -src=$(pwd)/swaggerui
