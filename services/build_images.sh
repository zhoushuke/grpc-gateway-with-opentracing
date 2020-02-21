#!/bin/bash

readonly -a arr=(a c)
readonly tag=1.5.2

for i in ${arr[@]}
do
  cp -f Dockerfile service-${i}
  pushd service-${i}
  CGO_ENABLED=0 go build  -ldflags '-w -s' -a -installsuffix cgo -o hello .
  docker build -t garystafford/go-srv-${i}:${tag} . --no-cache
  rm -rf Dockerfile
  popd
done

docker image ls | grep 'garystafford/go-srv-'
