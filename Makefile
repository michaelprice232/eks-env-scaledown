KUBE_CONTEXT ?= docker-desktop

test:
	go test -v ./...

lint:
	golangci-lint run ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# End-to-end tests. Spin up a local kind cluster via Terratest/Terraform and assert against it.
# Requires a running Docker engine (e.g. Docker Desktop) and Terraform and Kind CLIs. See test/e2e/readme.md.
test-e2e:
	cd test/e2e && go test -v -tags e2e -timeout 20m ./...

scale-up:
	SCALE_ACTION=ScaleUp KUBE_CONTEXT=$(KUBE_CONTEXT) go run cmd/main.go

scale-down:
	SCALE_ACTION=ScaleDown KUBE_CONTEXT=$(KUBE_CONTEXT) go run cmd/main.go

build-docker:
	docker buildx build --platform linux/amd64,linux/arm64 -t eks-env-scaledown .