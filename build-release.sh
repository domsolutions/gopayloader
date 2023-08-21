#!/bin/bash

cd build
rm -rf *.tar.gz && rm -rf gopayloader-*

cd ..
GOOS=windows go build -o ./build/gopayloader-windows-amd64.exe ./ && tar -czvf ./build/gopayloader-windows-amd64.tar.gz ./build/gopayloader-windows-amd64.exe
GOOS=linux go build -o ./build/gopayloader-linux-amd64 ./ && tar -czvf ./build/gopayloader-linux-amd64.tar.gz ./build/gopayloader-linux-amd64
GOOS=darwin go build -o ./build/gopayloader-darwin-amd64 ./ && tar -czvf ./build/gopayloader-darwin-amd64.tar.gz ./build/gopayloader-darwin-amd64
rm -rf ./build/*-amd64 ./build/*-amd64.exe