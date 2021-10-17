#!/bin/bash

# USAGE: ./install.sh [version]
# will install the latest tool executable to your /usr/local/bin

set -e

echo "Installing..."

TMPDIR=${TMPDIR:-"/tmp"}
pushd "$TMPDIR" > /dev/null
  mkdir -p gitsnap
  pushd gitsnap > /dev/null;
    distro=$(if [[ "$(uname -s)" == "Darwin" ]]; then echo "osx"; else echo "linux"; fi)
    if [ -n "$1" ]
    then
      echo "Will download and install v$1"
      curl -sSL --fail -o gitsnap.zip "https://github.com/apiiro/kubescout/releases/download/v$1/gitsnap-$1-$distro.zip"
    else
      curl -s --fail https://api.github.com/repos/apiiro/kubescout/releases/latest | grep "browser_download_url.*$distro.zip" | cut -d : -f 2,3 | tr -d \" | xargs curl -sSL --fail -o gitsnap.zip
    fi
    unzip gitsnap.zip
    chmod +x gitsnap-*
    cp -f gitsnap-* /usr/local/bin/kubescout
  popd > /dev/null
  rm -rf gitsnap
popd > /dev/null

echo "Done: $(kubescout -v)"
