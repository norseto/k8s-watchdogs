# k8s-watchdogs
Simple watchdogs for Kubernetes

## Evicted Pod Cleaner
This CronJob cleans "Evicted" pods.

### Installation
```
kubectl apply -f https://github.com/norseto/k8s-watchdogs/releases/download/evicted-cleaner-v0.1.2/evicted-cleaner.yaml
```

## Pod Rebalancer
Delete a pod that is scheduled to be biased to a specific node.

### Installation
```
kubectl apply -f https://github.com/norseto/k8s-watchdogs/releases/download/pod-rebalancer-v0.1.2/pod-rebalancer.yaml
```

### Limitation
Ignores pods with affinity or tolerations.

## Watchdogs CLI
Watchdogs CLI provides utility commands for Kubernetes maintenance.

### Usage
```
watchdogs [command]
```

Available Commands:
- `clean-evicted`: Clean evicted pods
- `rebalance-pods`: Rebalance pods across nodes
- `delete-oldest`: Delete oldest pods in a namespace
- `restart-deploy`: Restart deployment
- `restart-sts`: Restart statefulset

### Examples
Restart a Deployment:
```
watchdogs restart-deploy -n default my-deployment
```

Restart all Deployments in a namespace:
```
watchdogs restart-deploy -n default --all
```
or
```
watchdogs restart-deploy -n default -a
```

Restart a StatefulSet:
```
watchdogs restart-sts -n default my-statefulset
```

Restart all StatefulSets in a namespace:
```
watchdogs restart-sts -n default --all
```
or
```
watchdogs restart-sts -n default -a
```
