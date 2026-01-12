#! /usr/bin/env bash

sudo chown -R $(id -u):$(id -g) $HOME/.codex $HOME/.claude \
  /home/vscode/.aws /home/vscode/.kube \
  /tmp/.gocache /tmp/.gomodcache /go

