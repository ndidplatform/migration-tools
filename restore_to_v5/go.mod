module github.com/ndidplatform/migration-tools/restore_to_v5

go 1.13

require (
	github.com/gogo/protobuf v1.3.1
	github.com/ndidplatform/migration-tools/protos_v5 v0.0.0
	github.com/tendermint/tendermint v0.32.1
)

replace github.com/ndidplatform/migration-tools/protos_v5 => ../protos_v5
