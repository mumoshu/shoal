.PHONY: build
build:
	go build -i -ldflags="-X github.com/mumoshu/shoal.Version=dev" ./cmd/shoal
	go build -o go-example ./examples/go
	cd ./examples/k8s-e2e && go build -o k8s-e2e-example .
