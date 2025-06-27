# k8s-watchdogs
Simple watchdogs tools for Kubernetes

### Installation
```
kubectl kustomize https://github.com/norseto/k8s-watchdogs/config/watchdogs > watchdogs.yaml
```
And edit the configuration in the CronJob.

### Usage
```
Kubernetes utilities that can cleanup evicted pod, re-balance pod or restart deployment and so on

Usage:
  watchdogs [flags]
  watchdogs [command]

Available Commands:
  clean-evicted  Clean evicted pods
  completion     Generate the autocompletion script for the specified shell
  delete-oldest  Delete oldest pod(s)
  help           Help about any command
  rebalance-pods Delete bias scheduled pods
  restart-deploy Restart deployments by name or all with --all
  restart-sts    Restart statefulsets by name or all with --all

Flags:
  -h, --help   help for watchdogs
```

### clean-evicted
```
watchdogs clean-evicted --help
Clean evicted pods

Usage:
  watchdogs clean-evicted [flags]

Flags:
  -h, --help               help for clean-evicted
  -n, --namespace string   namespace
```

### delete-oldest
```
watchdogs delete-oldest --help
Delete oldest pod(s)

Usage:
  watchdogs delete-oldest [flags]

Flags:
  -h, --help               help for delete-oldest
  -m, --minPods int        Min pods required. (default 3)
  -n, --namespace string   namespace
  -p, --prefix string      Pod name prefix to delete.
```

### rebalance-pods
```
watchdogs rebalance-pods --help
Delete bias scheduled pods

Usage:
  watchdogs rebalance-pods [flags]

Flags:
  -h, --help               help for rebalance-pods
  -n, --namespace string   namespace
```

### restart-deploy
```
watchdogs restart-deploy --help
Restart one or more deployments by specifying deployment-name(s), or use --all to restart all in the namespace.

Usage:
  watchdogs restart-deploy [deployment-name|--all] [flags]

Flags:
  -a, --all                Restart all deployments in the namespace
  -h, --help               help for restart-deploy
  -n, --namespace string   namespace
```

### restart-sts
```
watchdogs restart-sts --help
Restart one or more statefulsets by specifying statefulset-name(s), or use --all to restart all in the namespace.

Usage:
  watchdogs restart-sts [statefulset-name|--all] [flags]

Flags:
  -a, --all                Restart all statefulsets in the namespace
  -h, --help               help for restart-sts
  -n, --namespace string   namespace
```

### Contribution Guide
For a short introduction to the repository layout and development workflow, see [docs/code_overview.md](docs/code_overview.md).

