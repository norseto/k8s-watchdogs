#!/bin/bash -xe

VERSION=$(grep 'RELEASE_VERSION\s*=' version.go  | awk -F= '{print $2}' | sed -e 's_"__g' -e 's/[[:space:]]//g')

if [[ ! "${VERSION}" =~ ^([0-9]+[.][0-9]+)[.]([0-9]+)(-(alpha|beta)[.]([0-9]+))?$ ]]; then
  echo "Version ${VERSION} must be 'X.Y.Z', 'X.Y.Z-alpha.N', or 'X.Y.Z-beta.N'"
  exit 1
fi

MINOR=${BASH_REMATCH[1]}
RELEASE_BRANCH="release-${MINOR}"

if [ "$(git tag -l "v${VERSION}")" ]; then
  echo "Tag v${VERSION} already exists"
  exit 0
fi

git tag -a -m "Release ${VERSION}" "v${VERSION}"
git push origin "v${VERSION}"

if [[ ! "${VERSION}" =~ .0-beta.1$ ]]; then
  exit 0
fi

git branch "${RELEASE_BRANCH}"
git push origin "${RELEASE_BRANCH}"
