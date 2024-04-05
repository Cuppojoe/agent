.PHONY: build-image-amd64 build-image-arm64 push-image build-proto
TAG ?= latest
REPOSITORY ?= 981027986651.dkr.ecr.us-east-2.amazonaws.com/ohm-fox
amd64:
	docker build --build-arg GOOS=linux --build-arg GOARCH=amd64 -t ${REPOSITORY}:${TAG}-amd64 .
arm64:
	docker build --build-arg GOOS=linux --build-arg GOARCH=arm64 -t ${REPOSITORY}:${TAG}-arm64 .
push: amd64 arm64
	docker push ${REPOSITORY}:${TAG}-amd64
	docker push ${REPOSITORY}:${TAG}-arm64
create-manifest-list: push
	docker manifest rm "${REPOSITORY}:${TAG}" || true
	docker manifest create --amend ${REPOSITORY}:${TAG} \
      	${REPOSITORY}@$(shell docker inspect ${REPOSITORY}:${TAG}-amd64 | jq -r '.[0].RepoDigests[0]' | awk -F '@' '{print $$2}') \
  		${REPOSITORY}@$(shell docker inspect ${REPOSITORY}:${TAG}-arm64 | jq -r '.[0].RepoDigests[0]' | awk -F '@' '{print $$2}')
	docker manifest annotate --arch arm64 ${REPOSITORY}:${TAG} ${REPOSITORY}@$(shell docker inspect ${REPOSITORY}:${TAG}-arm64 | jq -r '.[0].RepoDigests[0]' | awk -F '@' '{print $$2}')
push-manifest-list: create-manifest-list
	docker manifest push ${REPOSITORY}:${TAG}
build-proto:
	cd src \
	 && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protobuf/agent.proto \
	 && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protobuf/reflection.proto
