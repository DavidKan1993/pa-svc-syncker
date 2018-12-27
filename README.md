[![Build Status](https://travis-ci.org/inwinstack/pa-svc-syncker.svg?branch=master)](https://travis-ci.org/inwinstack/pa-svc-syncker) [![Docker Build Status](https://img.shields.io/docker/build/inwinstack/pa-svc-syncker.svg)](https://hub.docker.com/r/inwinstack/pa-svc-syncker/) [![codecov](https://codecov.io/gh/inwinstack/pa-svc-syncker/branch/master/graph/badge.svg)](https://codecov.io/gh/inwinstack/pa-svc-syncker) ![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)
# PA Kubernetes Service Syncker
The PA Syncker for Kubernetes provides automation synchronizing definitions for Kubernetes services to set the PA NAT, Security and Service.

## Building from Source
Clone repo into your go path under `$GOPATH/src`:
```sh
$ git clone https://github.com/inwinstack/pa-svc-syncker.git $GOPATH/src/github.com/inwinstack/pa-svc-syncker
$ cd $GOPATH/src/github.com/inwinstack/pa-svc-syncker
$ make dep
$ make
```

## Debug out of the cluster
Run the following command to debug:
```sh
$ go run cmd/main.go \
    -v=2 \
    --logtostderr \
    --kubeconfig=$HOME/.kube/config \
    --ignore-namespaces=kube-system,default,kube-public 
```

## Deploy in the cluster
Run the following command to deploy the controller:
```sh
$ kubectl apply -f deploy/
$ kubectl -n kube-system get po -l app=pa-svc-syncker
```
