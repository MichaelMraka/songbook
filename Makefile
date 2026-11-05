APP := songbook.app
DOCKER_IMAGE ?= docker.io/5keeve/pocketbook-go-sdk:6.3.0-b288-v1
CONTAINER ?= $(shell command -v docker >/dev/null 2>&1 && echo docker || echo podman)

.PHONY: test build clean

test:
	go test ./internal/...

build:
	$(CONTAINER) run --rm -v "$(CURDIR):/app:Z" -w /app $(DOCKER_IMAGE) \
		build -tags pocketbook -o $(APP) ./cmd/songbook

clean:
	rm -f $(APP)