#!/bin/sh

protoc -I=./data --go_out=./data ./data/data_v10.proto
protoc -I=./tendermint --go_out=./tendermint ./tendermint/tendermint_v10.proto
protoc -I=./param --go_out=./param ./param/param_v10.proto