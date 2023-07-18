# PWOD - Pilot WithOut Dataplane for Istio

## Requirements

- docker or podman
- kind
- kubectl
- istioctl
- [kwokctl](https://kwok.sigs.k8s.io/docs/user/installation/)

## Set up Cluster

``` bash
kwokctl create cluster --runtime kind
```

## Create Node

``` bash
kubectl apply -f https://kwok.sigs.k8s.io/examples/node.yaml
```

## Deploy Istio

``` bash
istioctl install -y
```

## Migrate Controllers to Real Node

``` bash
kubectl patch deploy istiod -n istio-system --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
```

## Start pwod and watch the logs

``` bash
kubectl port-forward svc/istiod -n istio-system 15017:443 15010:15010
```

``` bash
go run ./cmd/pwod
```

## Test

``` bash
kubectl label namespace default istio-injection=enabled
kubectl apply -f https://raw.githubusercontent.com.zsm.io/istio/istio/master/samples/bookinfo/platform/kube/bookinfo.yaml
```

## Clean up

``` bash
kwokctl delete cluster
```
