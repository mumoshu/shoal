.PHONY: build
build:
	go build ./cmd/shoal
	go build -o go-example ./examples/go
	cd ./examples/k8s-e2e && go build -o k8s-e2e-example .
