package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/doc-ai/tensorio-models/api"
	"github.com/doc-ai/tensorio-models/server"
	"github.com/doc-ai/tensorio-models/storage"
	"github.com/doc-ai/tensorio-models/storage/memory"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"io"
	"net"
	"net/http"
	"strings"
)

func initJaeger() (opentracing.Tracer, io.Closer) {
	log.Println("initializing jaeger")
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("sampler before: ", cfg.Sampler)

	log.Println("throttler", cfg.Throttler)

	log.Println("queue size", cfg.Reporter.QueueSize)

	log.Println(cfg.Reporter)

	cfg.Sampler.Type = "const"
	cfg.Sampler.Param = 1.0

	//cfg.Reporter.LogSpans = true

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		log.Fatalln("jaeger init failed", err)
	}
	log.Println(cfg.Sampler)
	log.Println(tracer, closer)

	return tracer, closer
}

func startGrpcServer(apiServer api.RepositoryServer) {
	serverAddress := ":8080"
	log.Println("Starting grpc on:", serverAddress)
	tracer, closer := initJaeger()
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	lis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		log.Fatalln(err)
	}

	api.RegisterRepositoryServer(grpcServer, apiServer)

	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("over")
}

func startProxyServer() {
	grpcServerAddress := ":8080"
	serverAddress := ":8081"
	log.Println("Starting json-rpc on:", serverAddress)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := api.RegisterRepositoryHandlerFromEndpoint(ctx, mux, grpcServerAddress, opts)
	if err != nil {
		log.Fatalln(err)
	}

	err = http.ListenAndServe(serverAddress, mux)
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	/* BEGIN cli */
	// Backend specification
	Backends := map[string]func() storage.RepositoryStorage{
		"memory": memory.NewMemoryRepositoryStorage,
	}
	BackendKeys := make([]string, len(Backends))
	i := 0
	for k := range Backends {
		BackendKeys[i] = k
		i++
	}
	BackendChoices := strings.Join(BackendKeys, ",")

	var backendArg string
	backendUsage := fmt.Sprintf("Specifies the repository storage backend to be used; choices: %s", BackendChoices)
	flag.StringVar(&backendArg, "backend", "", backendUsage)

	flag.Parse()

	backend, exists := Backends[backendArg]
	if !exists {
		log.Fatalf("Unknown backend: %s. Choices are: %s", backendArg, BackendChoices)
	}
	/* END cli */

	repositoryBackend := backend()
	apiServer := server.NewServer(repositoryBackend)

	go startGrpcServer(apiServer)
	go startProxyServer()

	// sleep forever
	select {}
}
