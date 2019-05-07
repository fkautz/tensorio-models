GOPATH:=$(shell go env GOPATH)
GRPC_LIST:=$(shell go list -m -f "{{.Dir}}" github.com/grpc-ecosystem/grpc-gateway)
GRPC_GATEWAY_PROTO_DIR:="${GRPC_LIST}/third_party/googleapis"
TIMESTAMP:=$(shell date -u +%s)
RUN_ARGS=-backend memory

default: fmt build

fmt:
	gofmt -w -s .

docker-build:
	docker build -t docai/tensorio-models -f dockerfiles/Dockerfile.repository .

run: docker-build
	docker run --env JAEGER_SERVICE_NAME=models --env JAEGER_AGENT_HOST=host.docker.internal -p 8080:8080 -p 8081:8081 docai/tensorio-models ${RUN_ARGS}
	#docker run --env JAEGER_SERVICE_NAME=models --env JAEGER_AGENT_HOST=host.docker.internal --env JAEGER_REPORTER_LOG_SPANS=true -p 8080:8080 -p 8081:8081 docai/tensorio-models ${RUN_ARGS}
	#docker run --env JAEGER_SERVICE_NAME=models --env JAEGER_ENDPOINT=http://host.docker.internal:16686/api/traces/ --env JAEGER_REPORTER_LOG_SPANS=true -p 8080:8080 -p 8081:8081 docai/tensorio-models ${RUN_ARGS}

api/repository.pb.go: api/repository.proto
	cd api && protoc -I . repository.proto --go_out=plugins=grpc:. --proto_path=${GOPATH}/src --proto_path=$(GOPATH)/pkg/mod --proto_path=$(GRPC_GATEWAY_PROTO_DIR)

api/repository.pb.gw.go: api/repository.proto
	cd api && protoc -I . repository.proto --grpc-gateway_out=logtostderr=true:. --proto_path=$(GOPATH)/src --proto_path=$(GOPATH)/pkg/mod --proto_path=$(GRPC_GATEWAY_PROTO_DIR)

api/repository.swagger.json: api/repository.proto
	cd api && protoc -I . repository.proto --swagger_out=logtostderr=true:. --proto_path=$(GOPATH)/src --proto_path=$(GOPATH)/pkg/mod --proto_path=$(GRPC_GATEWAY_PROTO_DIR)

build: api/repository.pb.go api/repository.pb.gw.go api/repository.swagger.json
	go test ./... -cover
	go build ./...

coverage: api/repository.pb.go api/repository.pb.gw.go api/repository.swagger.json
	go test -coverprofile=test.out ./...
	go tool cover -html=test.out -o coverage-$(TIMESTAMP).html
	echo "Coverage report: coverage-$(TIMESTAMP).html"

coverage-cleanup:
	rm test.out coverage-*.html

jaeger-start:
	docker run -d --name jaeger   -e COLLECTOR_ZIPKIN_HTTP_PORT=9411   -p 5775:5775/udp   -p 6831:6831/udp   -p 6832:6832/udp   -p 5778:5778   -p 16686:16686   -p 14268:14268   -p 9411:9411   jaegertracing/all-in-one:1.11
	echo "Access jaeger at: http://localhost:16686/search"

jaeger-stop:
	docker kill "/jaeger" ; true
	docker rm "/jaeger" ; true
