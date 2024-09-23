#!/bin/bash

go mod tidy
CGO_ENABLED=0 go build -ldflags "-w -s" -trimpath