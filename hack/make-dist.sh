#!/bin/bash
make build-installer IMG=docker.io/norseto/oci-lb-registrar:v$(hack/get-version.sh)
