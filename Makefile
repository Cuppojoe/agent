.PHONY: build-image-amd64 build-image-arm64 push-image build-proto
TAG ?= 0.9.6
IMAGE ?= ghcr.io/Cuppojoe/agent
CTX   ?= .
# Space-separated list of *extra* tags (e.g. "v1 v1.2 latest"); may be empty
TAGS  ?=

push-image-amd64:
	docker buildx build --platform linux/amd64 -t $(IMAGE):$(TAG)-amd64 --push $(CTX)

push-image-arm64:
	docker buildx build --platform linux/arm64 -t $(IMAGE):$(TAG)-arm64 --push $(CTX)

# Stitch per-arch images into a multi-arch manifest; add extra tags when provided
combine-push-images:
	docker buildx imagetools create \
	  -t $(IMAGE):$(TAG) \
	  $(foreach t,$(TAGS),-t $(IMAGE):$(t) ) \
	  $(IMAGE):$(TAG)-amd64 \
	  $(IMAGE):$(TAG)-arm64

build-proto:
	cd src \
	 && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protobuf/agent.proto \
	 && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protobuf/reflection.proto
