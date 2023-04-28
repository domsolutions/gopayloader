#!/bin/bash

GOOS=windows go build -o gopayloader-windows-amd64.exe ./
GOOS=linux go build -o gopayloader-linux-amd64 ./
GOOS=darwin go build -o gopayloader-darwin-amd64 ./