### Launch configuration
---
Launch configuration "Microk8s" in launch.json requires microk8s.config file in this directory. To create this file,
1. Run `kubectl config view --raw` and save as microk8s.config.
1. Run `multipass list` and get microk8s IP address.
1. Rewite server IP address in microk8s.config.

