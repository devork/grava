# The folder of the make file
ROOT := $(patsubst %/,%,$(dir $(realpath $(lastword $(MAKEFILE_LIST)))))
GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
LDFLAGS := "-X main.sha=`git log -1 --pretty=format:%H` -X main.date=`date -u +%Y%m%d%H%M%S` -X main.version=1.0.0"

clean:
	go clean

deps:
	glide install

build:
	go clean
	go build -ldflags ${LDFLAGS} -o gravad

dockerlin:
	go clean
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o gravad
	docker build -t gravad:latest -f dockerfile .