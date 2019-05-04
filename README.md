# hello-micro

Microservices notes / hello world implementation.  Inspired by a
couple of different tutorials, plus go-micro and ja-micro.  See references 
for some of the ones I found useful.

# features

Core features:
- gRPC and HTTP/2 server         (DONE)
- test client                    (DONE)
- OpenAPI and web interface      (in progress)

Extra features:
- optional persistence to SQL database (done) or Mongo
- OpenTracing                    (DONE)
- GraphQL
- Prometheus metrics
- retains history of greetings?  (DONE)

# Running

    ./scripts/regenerate.sh

    go run server/main.go
    env HELLO_GRPC_PORT=:50051 go run server/main.go
    go run client/main.go

    grpcurl --import-path api --proto hello.proto list
    grpcurl --import-path api --proto hello.proto describe hello.Greeter   
    grpcurl --plaintext -d '{"name": "world"}' --proto api/hello.proto localhost:50051 hello.Greeter.SayHello
    grpcurl --plaintext -d '{"name": "world"}' --proto api/hello.proto localhost:50051 hello.Greeter.LotsOfReplies
    http POST localhost:8080/v1/example/echo name=World
    http --stream POST localhost:8080/v1/example/lots name=World  # to get back streaming results

# References

- https://medium.com/@vptech/complexity-is-the-bane-of-every-software-engineer-e2878d0ad45a
