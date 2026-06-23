# Build targets
BINARY_NAME=mean

.PHONY: all build run clean test install

all: build

build:
	go build -o $(BINARY_NAME).exe ./cmd/mean

run: build
	./$(BINARY_NAME).exe

clean:
	go clean
	if exist $(BINARY_NAME).exe del /F /Q $(BINARY_NAME).exe

test:
	go test -v ./...

install: build
	go install ./cmd/mean
