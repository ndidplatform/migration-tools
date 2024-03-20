#!/bin/sh

protoc -I=./data --go_out=./data ./data/data_v8.proto
protoc -I=./tendermint --go_out=./tendermint ./tendermint/tendermint_v8.proto
protoc -I=./param --go_out=./param ./param/param_v8.proto