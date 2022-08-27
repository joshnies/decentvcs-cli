#!/bin/sh

# Make sure git is installed
if ! [ -x "$(command -v git)" ]; then
  echo 'Error: git is not installed.' >&2
  exit 1
fi

# Install CLI
go build && go install

# Rename binary
mv ~/go/bin/cli ~/go/bin/dvcs

echo "Installed"
