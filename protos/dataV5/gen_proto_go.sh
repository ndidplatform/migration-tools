#!/bin/sh

protoc -I=./ --go_out=./ ./dataV5.proto