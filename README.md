# k8s-watchdogs
Simple watchdogs for Kubernetes

## Evicted Pod Cleaner
This CronJos cleans "Evicted" pods.

### Installation
```
kubectl apply -f https://github.com/norseto/k8s-watchdogs/releases/download/evicted-cleaner-v0.1.0/evicted-cleaner.yaml
```

## Pod Rebalancer
Delete a pod that is scheduled to be biased to a specific node.

### Installation
```
kubectl apply -f https://github.com/norseto/k8s-watchdogs/releases/download/pod-rebalancer-v0.1.0/pod-rebalancer.yaml
```

### Limitation
Ignores pods with affinity or tolerations.
