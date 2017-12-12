
.PHONY: build
build:
	go build -o bin/k8s-gateway ./cmd/...

# To use docker-build, you need to have Docker installed and configured. You should also set
# DOCKER_REGISTRY to your own personal registry if you are not pushing to the official upstream.
.PHONY: docker-build
docker-build:
	GOOS=linux GOARCH=amd64 go build -o rootfs/k8s-gateway ./cmd/...
	docker build -t deis/brigade-k8s-gateway:latest .

# You must be logged into DOCKER_REGISTRY before you can push.
.PHONY: docker-push
docker-push:
	docker push deis/brigade-k8s-gateway
