#!/bin/bash

go build  -ldflags="-s -w" -o mds ./bin/server
go build  -ldflags="-s -w" -o mdc ./bin/client
upx mds mdc
