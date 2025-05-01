#!/bin/bash

AGE_VERSION="v1.2.1"
SOPS_VERSION="v3.10.2"


if [ -d "$HOME/bin" ]; then
    INSTALL_DIR="$HOME/bin"
else
    INSTALL_DIR="/usr/local/bin"
fi

# Detect: Darwin or Linux; amd64 or arm or arm64
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
if [ "$ARCH" == "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" == "aarch64" ]; then
    ARCH="arm64"
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf $TMPDIR' EXIT

# Download age
AGE_URL="https://github.com/FiloSottile/age/releases/download/${AGE_VERSION}/age-${AGE_VERSION}-${OS}-${ARCH}.tar.gz"

curl -sL $AGE_URL | tar -C $TMPDIR -xzf -
mv $TMPDIR/age $INSTALL_DIR

# Download sops
SOPS_URL="https://github.com/getsops/sops/releases/download/${SOPS_VERSION}/sops-${SOPS_VERSION}.${OS}.${ARCH}"

curl -sL $SOPS_URL -o $TMPDIR/sops
chmod +x $TMPDIR/sops
mv $TMPDIR/sops $INSTALL_DIR

echo "age and sops installed to $INSTALL_DIR"

# TODO: set up a sops config? or will it be checked into repo

