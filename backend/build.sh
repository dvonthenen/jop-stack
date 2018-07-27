#! /usr/bin/env bash

rm -rf ./vendor
rm glide.lock
rm -f ./backend
glide up

grep -R --exclude-dir vendor --exclude-dir .git --exclude build.sh TODO ./

#GOOS=linux GOARCH=amd64 go build .
docker run --rm -e GOPATH=/usr/src/go -v ~/go:/usr/src/go -w /usr/src/go/src/github.com/dvonthenen/jop-stack/backend golang:1.10.2-alpine3.7 go build -v

docker build -t dvonthenen/jop-backend .
docker tag dvonthenen/jop-backend dvonthenen/jop-backend:v0.0.3
docker push dvonthenen/jop-backend:v0.0.3
