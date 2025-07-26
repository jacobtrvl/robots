
.PHONY: build
build:
	go build -o bin/$(BINARY_NAME) ./cmd/rest
	@echo "Build complete: bin/$(BINARY_NAME)"

.PHONY: run
run:
	go run ./cmd/rest


