#!/bin/bash -xe

if [ $# -lt 1 ]; then
  echo "Usage: $0 NEW_VERSION" 1>&2
  exit 1
fi

PAT_VERSION='^([0-9]+[.][0-9]+)[.]([0-9]+|0-(alpha|beta)[.][1-9][0-9]*)$'

PAT_ALPHA='(^[0-9]+[.][0-9]+[.]0+)-alpha[.][1-9][0-9]*$'
PAT_ALPHA_OR_BETA='^[0-9]+[.][0-9]+[.]0-(alpha|beta)[.][1-9][0-9]*$'
PAT_RELEASE='^([0-9]+[.][0-9]+)[.][0-9]+$'
PAT_BETA_OR_RELEASE='(^[0-9]+[.][0-9]+[.])([0-9]+|0-beta[.][1-9][0-9]*)$'

CURRENT_VERSION=$(grep 'RELEASE_VERSION[[:space:]]*=' version.go  | awk -F= '{print $2}' | sed -e 's_"__g' -e 's/[[:space:]]//g')

if [[ ! "${CURRENT_VERSION}" =~ ${PAT_VERSION} ]]; then
  echo "Current version ${CURRENT_VERSION} is invalid. It must be 'X.Y.Z', 'X.Y.0-alpha.N' or 'X.Y.0-beta.N'"
  exit 1
fi
CURRENT_MINOR=${BASH_REMATCH[1]}

NEW_VERSION=$1
if [[ ! "${NEW_VERSION}" =~ ${PAT_VERSION} ]]; then
  echo "New version ${NEW_VERSION} must be 'X.Y.Z', 'X.Y.0-alpha.N' or 'X.Y.0-beta.N'"
  exit 1
fi

NEW_MINOR=${BASH_REMATCH[1]}

# Alpha or Beta version should be derived from
# 1. alpha version (ex. 1.0.0-alpha.1 -> 1.0.0-alpha.2, 0.9.0 -> 1.0.0-alpha.1)
# 2. first beta version.
if [[ "${NEW_VERSION}" =~ ${PAT_ALPHA_OR_BETA} ]] && [[ ! "${CURRENT_VERSION}" =~ ${PAT_ALPHA_OR_BETA} ]] ; then
  echo "Cannot set alpha or beta version ${NEW_VERSION} on release branch(${CURRENT_VERSION})"
  exit 1
fi

# A release version can set only beta or release version.
if [[ "${NEW_VERSION}" =~ ${PAT_RELEASE} ]] && [[ ! "${CURRENT_VERSION}" =~ ${PAT_BETA_OR_RELEASE} ]] ; then
  echo "Cannot set release version ${NEW_VERSION} on ${CURRENT_VERSION}"
  exit 1
fi

# Only Alpha or version can set next minor version.
if [[ ! "${NEW_VERSION}" =~ ${PAT_ALPHA} ]] && [ "${CURRENT_MINOR}" != "${NEW_MINOR}" ] ; then
  echo "Cannot set different minor version ${NEW_VERSION} on ${CURRENT_VERSION}"
  exit 1
fi

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BASE_DIR=${SCRIPT_DIR}/..

sed -i.bak -e "s@newTag:[[:space:]]*v${CURRENT_VERSION}@newTag: v${NEW_VERSION}@g" ${BASE_DIR}/config/manager/kustomization.yaml
sed -i.bak -e "s@RELEASE_VERSION[[:space:]]*=[[:space:]]*\"${CURRENT_VERSION}\"@RELEASE_VERSION = \"${NEW_VERSION}\"@g" ${BASE_DIR}/version.go

make build-installer IMG=norseto/oci-lb-registrar:v$(hack/get-version.sh)
