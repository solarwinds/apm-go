package main

import (
	"context"
	"github.com/solarwinds/apm-go/swo"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"time"
)

func main() {
	// Initialize the SolarWinds APM library
	cb, err := swo.Start(
		// Optionally add service-level resource attributes
		semconv.ServiceName("my-service"),
		semconv.ServiceVersion("v0.0.1"),
		attribute.String("environment", "testing"),
	)
	if err != nil {
		// Handle error
	}
	// This function returned from `Start()` will tell the apm library to
	// shut down, often deferred until the end of `main()`.
	defer cb()
	lis, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(
		// We provide `otelgrpc` interceptors to add instrumentation to each call
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		// Even though we have no streaming calls in this service, adding this
		// instrumentation adds zero overhead. It's a good idea to _always_ have
		// both Unary and Stream interceptors for both server and client.
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)
	RegisterPingerServer(grpcServer, newPingServer())

	// Create a loop that calls the pinger via an instrumented grpc client
	go func() {
		// Here we create an instrumented client
		conn, err := grpc.Dial("localhost:9090",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			// otelgrpc instrumentation for the client
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
		)
		if err != nil {
			panic(err.Error())
		}
		defer conn.Close()
		client := NewPingerClient(conn)
		for {
			time.Sleep(1 * time.Second)
			log.Println("Ping!")
			_, err := client.Ping(context.Background(), &PingRequest{})
			if err != nil {
				panic(err.Error())
			}
		}
	}()
	grpcServer.Serve(lis)
}

type pinger struct {
	UnimplementedPingerServer
}

func (p pinger) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	log.Println("Pong!")
	return &PingResponse{}, nil
}

var _ PingerServer = pinger{}

func newPingServer() PingerServer {
	return pinger{}
}
