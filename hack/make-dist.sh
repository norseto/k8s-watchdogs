#!/bin/bash
make build-installer IMG=norseto/oci-lb-registrar:v$(hack/get-version.sh)
