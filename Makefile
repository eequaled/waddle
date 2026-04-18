.PHONY: dev build build-llama build-onnx test test-storage test-synthesis clean

# Default: build without optional native deps
build:
	go build -p 1 ./...

# Build with llama.cpp support (now purego, no tags needed)
build-llama:
	go build -p 1 ./...

# Build with ONNX Runtime support
build-onnx:
	go build -p 1 -tags onnx ./...

# Build with both native deps
build-full:
	go build -p 1 -tags onnx ./...

# Dev server
dev:
	wails dev

# Full production build
dist:
	wails build

# Tests
test:
	go test -p 1 ./pkg/...

test-storage:
	go test -p 1 ./pkg/storage/ -v

test-synthesis:
	go test -p 1 ./pkg/synthesis/ -v

# Frontend
frontend-build:
	cd frontend && npm run build

frontend-test:
	cd frontend && npm run test

frontend-lint:
	cd frontend && npm run lint

clean:
	go clean ./...
