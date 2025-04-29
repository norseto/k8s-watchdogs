#!/bin/bash
VERSION=$(grep 'RELEASE_VERSION\s*=' version.go  | awk -F= '{print $2}' | sed -e 's_"__g' -e 's/[[:space:]]//g')
echo ${VERSION}
