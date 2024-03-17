.PHONY: roles
roles:
	controller-gen rbac:roleName=k8s-watchdogs-role paths=./... output:dir=config/watchdogs/rbac/
