#! /usr/bin/env bash

sudo chown -R $(id -u):$(id -g) $HOME/.codex $HOME/.claude \
  /home/vscode/.aws /home/vscode/.kube \
  /home/vscode/.gocache /home/vscode/.gomodcache /go
