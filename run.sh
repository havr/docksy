#!/bin/bash

CONTAINER_DOCKER_CERTS=/docker-certs
CONTAINER_CERTS=/certs

DOCKSY_CONTAINER=docksy
ETCD=http://192.168.99.100:4001
DOCKER_HOST=tcp://192.168.99.100:2376
DOCKER_CERTS=/Users/havr/.docker/machine/machines/default
CERTS=/Users/havr/Temp

docker run -i -t --name $DOCKSY_CONTAINER -v $CERTS:$CONTAINER_CERTS -v $DOCKER_CERTS:$CONTAINER_DOCKER_CERTS -p 80:80 -p 443:443 docksy /go/bin/main --etcd=$ETCD --docker=$DOCKER_HOST --docker-certs=$CONTAINER_DOCKER_CERTS --certs=$CONTAINER_CERTS