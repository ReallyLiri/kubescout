#!/bin/bash

# USAGE: ./install.sh [version]
# will install the latest tool executable to your /usr/local/bin

set -e

echo "Installing..."

TMPDIR=${TMPDIR:-"/tmp"}
pushd "$TMPDIR" > /dev/null
  mkdir -p kubescout
  pushd kubescout > /dev/null;
    distro=$(if [[ "$(uname -s)" == "Darwin" ]]; then echo "osx"; else echo "linux"; fi)
    if [ -n "$1" ]
    then
      echo "Will download and install v$1"
      curl -sSL --fail -o kubescout.zip "https://github.com/reallyliri/kubescout/releases/download/v$1/kubescout-$distro.zip"
    else
      curl -s --fail https://api.github.com/repos/reallyliri/kubescout/releases/latest | grep "browser_download_url.*$distro.zip" | cut -d : -f 2,3 | tr -d \" | xargs curl -sSL --fail -o kubescout.zip
    fi
    unzip kubescout.zip
    chmod +x kubescout
    cp -f kubescout /usr/local/bin/kubescout
  popd > /dev/null
  rm -rf kubescout
popd > /dev/null

echo "Done: $(kubescout -v)"
