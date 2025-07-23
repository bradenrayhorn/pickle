#!/bin/bash

set -e

if [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
	sudo apt-get -yq update
	sudo apt-get -yq install libgtk-3-0 libwebkit2gtk-4.1-dev gcc-aarch64-linux-gnu
fi

version=$(grep "github.com/wailsapp/wails" go.mod | awk '{print $2}')

echo Installing wails $version
go install github.com/wailsapp/wails/v2/cmd/wails@$version

