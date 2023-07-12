#!make

GOCC=go
MKDIR_P=mkdir -p

BIN_PATH=./build
BIN="./build/goteth"

.PHONY: check build install run clean

build: 
	$(GOCC) build -o $(BIN)

install:
	$(GOCC) install

clean:
	rm -r $(BIN_PATH)

