#!/bin/bash

go get github.com/robertkrimen/godocdown/godocdown
$GOPATH/src/github.com/robertkrimen/godocdown/godocdown/godocdown github.com/polyverse-security/appconfig > README.md
echo Done. README.md has been refreshed.
