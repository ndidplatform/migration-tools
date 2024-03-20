#!/bin/sh

protoc -I=./data --go_out=./data ./data/data_v7.proto
protoc -I=./tendermint --go_out=./tendermint ./tendermint/tendermint_v7.proto
protoc -I=./param --go_out=./param ./param/param_v7.proto