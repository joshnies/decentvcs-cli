#!/bin/sh

# Make sure git is installed
if ! [ -x "$(command -v git)" ]; then
  echo 'Error: git is not installed.' >&2
  exit 1
fi

# Install CLI
go build && go install

echo "Installed"

# Add alias if not set
if [[ $SHELL = "/bin/zsh" ]]; then
  # zsh
  shell_file="$HOME/.zshrc"
elif [[ $SHELL = "/bin/bash" ]]; then
  # bash
  if test -f ~/.profile; then
    shell_file="$HOME/.profile"
  else
    shell_file="$HOME/.bash_profile"
  fi
else
  echo "Warning: Unknown shell $SHELL; aliases were not added"
  exit 0
fi

if ! grep -q "alias dvcs" $shell_file; then
  echo 'alias dvcs="decent vcs"' >> $shell_file
  echo "Added the following aliases to $shell_file:"
  echo '  alias dvcs="decent vcs"'
fi
