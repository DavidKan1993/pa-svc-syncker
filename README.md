[![Build Status](https://travis-ci.org/inwinstack/pa-svc-syncker.svg?branch=master)](https://travis-ci.org/inwinstack/pa-svc-syncker) [![Docker Build Status](https://img.shields.io/docker/build/inwinstack/pa-svc-syncker.svg)](https://hub.docker.com/r/inwinstack/pa-svc-syncker/) [![codecov](https://codecov.io/gh/inwinstack/pa-svc-syncker/branch/master/graph/badge.svg)](https://codecov.io/gh/inwinstack/pa-svc-syncker) ![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)
# PA Kubernetes Service Syncker
A controller automatically sync Kubernetes service to set NAT and Security policy.

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
    --kubeconfig=$HOME/.kube/config \
    --logtostderr \
    --ignore-namespaces=kube-system,default,kube-public \
    --log-setting=siem_forward \
    --group=inwin-monitor \
    -v=2
```

## Deploy in the cluster
Run the following command to deploy the controller:
```sh
$ kubectl apply -f deploy/
$ kubectl -n kube-system get po -l app=pa-svc-syncker
```
