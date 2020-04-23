#!/bin/sh

protoc -I=./ --go_out=./ ./dataV4.proto