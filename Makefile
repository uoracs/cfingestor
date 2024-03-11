.PHONY: build install clean run container handler build_x86

all: build

build:
    @go build -o bin/cfingestor cmd/cfingestor/main.go

build_x86:
	@GOOS=linux GOARCH=amd64 go build -o bin/cfingestor cmd/cfingestor/main.go

run: build
    @go run cmd/cfingestor/main.go

install: build
    @cp bin/cfingestor /usr/local/bin/cfingestor

container:
    @docker build -t cfingestor .

handler:
    @go build -o handler cmd/cfingestor/main.go

clean:
    @rm -f bin/cfingestor /usr/local/bin/cfingestor
