.PHONY: all build clean install release

all: build

build:
	go build -o bin/knocker ./cmd/knocker

clean:
	rm -rf bin dist

install:
	go install ./cmd/knocker

release:
	goreleaser release --snapshot --clean