#!/usr/bin/env bash

set -e

if [ -n "$TRAVIS_BUILD_DIR" ]; then
    cd $TRAVIS_BUILD_DIR 
else
	cd $GOPATH/src/github.com/thrasher-/gocryptotrader
fi

echo "" > testdata/coverage.txt

for d in $(go list ./... | grep -v vendor); do
    go test -race -coverprofile=profile.out -covermode=atomic -cover $d
    if [ -f profile.out ]; then
        cat profile.out >> testdata/coverage.txt
        rm profile.out
    fi
done
