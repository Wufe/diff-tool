#!/usr/bin/env bash
output_name="diff-tool"
package=github.com/wufe/diff-tool

env GOOS=windows GOARCH=386 go build -o bin/${output_name}-win386.exe $package
env GOOS=windows GOARCH=amd64 go build -o bin/${output_name}-amd64.exe $package
env GOOS=darwin GOARCH=amd64 go build -o bin/${output_name}-darwin $package