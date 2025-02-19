.PHONY: build-image-amd64 build-image-arm64 push-image build-proto
TAG ?= 0.9.4-arm
REPOSITORY ?= kylecupp/agent
push-image:
	docker buildx build --platform linux/arm64 -t ${REPOSITORY}:${TAG} --push .
build-proto:
	cd src \
	 && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protobuf/agent.proto \
	 && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protobuf/reflection.proto
