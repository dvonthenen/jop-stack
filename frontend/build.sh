#! /usr/bin/env bash

rm -rf ./vendor
rm glide.lock
rm -f ./frontend
glide up

grep -R --exclude-dir vendor --exclude-dir .git --exclude build.sh TODO ./

#GOOS=linux GOARCH=amd64 go build .
docker run --rm -e GOPATH=/usr/src/go -v ~/go:/usr/src/go -w /usr/src/go/src/github.com/dvonthenen/jop-stack/frontend golang:1.10.2-alpine3.7 go build -v

docker build -t dvonthenen/jop-frontend .
docker tag dvonthenen/jop-frontend dvonthenen/jop-frontend:v0.0.3
docker push dvonthenen/jop-frontend:v0.0.3
