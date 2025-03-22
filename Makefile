# Description: Makefile for building the docker image
DOCKER_TAG=latest
DOCKER_IMAGE=ghcr.io/simonschneider/chore-tracker
DOCKER=podman

build_linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/linux/app cmd/main.go

build_docker:
	$(DOCKER) build --platform=linux/amd64 -t $(DOCKER_IMAGE) .

push:
	$(DOCKER) tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):latest
	$(DOCKER) tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_TAG)
	$(DOCKER) push $(DOCKER_IMAGE):latest
	$(DOCKER) push $(DOCKER_IMAGE):$(DOCKER_TAG)

build: build_linux build_docker push
