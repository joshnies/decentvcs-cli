#!/bin/sh

# Make sure git is installed
if ! [ -x "$(command -v git)" ]; then
  echo 'Error: git is not installed.' >&2
  exit 1
fi

# Install CLI
go build && go install

echo "Installed"

# Add aliases if not set
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

# Add `dvcs` alias
if ! grep -q "alias dvcs" $shell_file; then
  dvcs_alias='alias dvcs="decent vcs"'
  echo $dvcs_alias >> $shell_file
  echo "Added the following aliases to $shell_file:"
  echo "  $dvcs_alias"
fi
