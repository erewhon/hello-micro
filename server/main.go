//go:generate protoc -I ../api --go_out=plugins=grpc:../api ../api/hello.proto

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"

	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/erewhon/hello-micro/api"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"google.golang.org/grpc"
)

// server is used to implement helloworld.GreeterServer.
type Server struct {
	db     *gorm.DB
	logger *zap.Logger
	log    *zap.SugaredLogger
}

type DBHelloRequest struct { // We embed gorm.model and HelloRequest so we can capture the request along with Gorm metadata
	gorm.Model
	pb.HelloRequest
}

// SayHello implements helloworld.GreeterServer
func (s *Server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	s.log.Infof("SayHello Received: %v", in.Name)
	s.store(in)

	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func (s *Server) LotsOfReplies(in *pb.HelloRequest, stream pb.Greeter_LotsOfRepliesServer) (err error) {
	s.log.Infof("LotsOfReplies Received: %v", in.Name)
	s.store(in)

	for i := 0; i < 10; i += 1 {
		response := &pb.HelloReply{Message: fmt.Sprintf("Hello %s %d", in.Name, i+1)}

		if err = stream.Send(response); err != nil {
			return
		}

		time.Sleep(2 * time.Second) // Dramatic pause
	}

	return nil
}

func (s *Server) store(in *pb.HelloRequest) {
	r := &DBHelloRequest{}
	r.HelloRequest = *in

	s.db.Create(r)
}

func (s *Server) Run() {

	s.log.Infow("Starting up Greater service")

	go func() {
		s.runGRPCServer()
	}()

	go func() {
		s.runGWServer()
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	s.log.Infow("Shutting down server")

	defer s.db.Close()
	defer s.logger.Sync() // flushes buffer, if any
}

func alwaysLogPayload(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
	return true
}

func (s *Server) runGRPCServer() {
	port := viper.GetString("grpc_port")
	lis, err := net.Listen("tcp", port)
	if err != nil {
		s.log.Fatalf("failed to listen: %v", err)
	}
	grpc := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_opentracing.StreamServerInterceptor(),
			//grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(s.logger),
			grpc_zap.PayloadStreamServerInterceptor(s.logger, alwaysLogPayload), // We may or may not want to do this...
			//grpc_auth.StreamServerInterceptor(myAuthFunction),
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			//grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(s.logger),
			grpc_zap.PayloadUnaryServerInterceptor(s.logger, alwaysLogPayload), // We may or may not want to do this...
			//grpc_auth.UnaryServerInterceptor(myAuthFunction),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)
	pb.RegisterGreeterServer(grpc, s)

	s.log.Infof("Starting gRPC server on port %s", port)

	if err := grpc.Serve(lis); err != nil {
		s.log.Fatalf("failed to serve: %v", err)
	}

}

func (s *Server) runGWServer() error {
	addr := viper.GetString("http_port")
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterGreeterHandlerFromEndpoint(ctx, mux, viper.GetString("grpc_port"), opts)
	if err != nil {
		return err
	}

	s.log.Infof("Starting Gateway server on %s", addr)

	return http.ListenAndServe(addr, mux)
}

func NewServer() (server *Server, err error) {

	server = &Server{}

	viper.SetEnvPrefix("hello")
	viper.AutomaticEnv()
	viper.SetDefault("http_port", ":8080")
	viper.SetDefault("grpc_port", ":9090")
	viper.SetDefault("db_conn", "root@/scratch?charset=utf8&parseTime=True&loc=Local")

	server.logger, _ = zap.NewProduction()
	server.log = server.logger.Sugar()

	server.db, err = gorm.Open("mysql", viper.GetString("db_conn"))
	if err != nil {
		return
	}

	server.db.AutoMigrate(&DBHelloRequest{})

	return server, nil
}

func main() {

	server, err := NewServer()
	if err != nil {
		panic(err)
	}

	server.Run()
}
