#! /usr/bin/env bash

sudo chown -R vscode:vscode \
  /home/vscode/.aws /home/vscode/.kube \
  /home/vscode/.gocache /go

sudo chown -R $(id -u):$(id -g) $HOME/.codex
