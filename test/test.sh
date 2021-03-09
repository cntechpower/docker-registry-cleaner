#!/bin/bash
set -e
docker run -d -p 5000:5000 -v `pwd`/registry.yml:/etc/docker/registry/config.yml --name registry registry:2

docker pull ubuntu:16.04
docker tag ubuntu:16.04 localhost:5000/ubuntu:16.04
docker push localhost:5000/ubuntu:16.04

docker pull ubuntu:18.04
docker tag ubuntu:18.04 localhost:5000/ubuntu:18.04
docker push localhost:5000/ubuntu:18.04

docker pull ubuntu:20.10
docker tag ubuntu:20.10 localhost:5000/ubuntu:20.10
docker push localhost:5000/ubuntu:20.10



# docker rm -f registry
