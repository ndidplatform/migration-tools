#!/bin/sh

gen_proto_go() {
  local VERSION=$1

  protoc -I=./did/${VERSION}/protos/data --go_out=./did/${VERSION}/protos/data ./did/${VERSION}/protos/data/data_${VERSION}.proto
  protoc -I=./did/${VERSION}/protos/tendermint --go_out=./did/${VERSION}/protos/tendermint ./did/${VERSION}/protos/tendermint/tendermint_${VERSION}.proto
}

gen_proto_go "v1"
gen_proto_go "v2"
gen_proto_go "v3"
gen_proto_go "v4"
gen_proto_go "v5"
