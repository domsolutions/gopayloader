#!/bin/bash

rm -rf *.tar.gz && rm -rf gopayloader-*

GOOS=windows go build -o gopayloader-windows-amd64.exe ./ && tar -czvf gopayloader-windows-amd64.tar.gz gopayloader-windows-amd64.exe
GOOS=linux go build -o gopayloader-linux-amd64 ./ && tar -czvf gopayloader-linux-amd64.tar.gz gopayloader-linux-amd64
GOOS=darwin go build -o gopayloader-darwin-amd64 ./ && tar -czvf gopayloader-darwin-amd64.tar.gz gopayloader-darwin-amd64