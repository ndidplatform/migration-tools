#!/bin/sh

protoc -I=./ --go_out=./ ./tendermint.proto