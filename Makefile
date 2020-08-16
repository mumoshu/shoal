.PHONY: build
build:
	go build ./cmd/shoal
	go build -o go-example ./examples/go
