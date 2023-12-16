#!/bin/sh
# Set k8s api version

K8SVERSION=1.20.2
LIBS="client-go api apimachinery"

for lib in ${LIBS}
do
	go get k8s.io/${lib}@kubernetes-${K8SVERSION}
done
