KUBE_CONTEXT ?= docker-desktop

test:
	go test -v ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

scale-up:
	SCALE_ACTION=ScaleUp KUBE_CONTEXT=$(KUBE_CONTEXT) go run cmd/main.go

scale-down:
	SCALE_ACTION=ScaleDown KUBE_CONTEXT=$(KUBE_CONTEXT) go run cmd/main.go