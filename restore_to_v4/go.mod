module github.com/ndidplatform/migration-tools/restore_to_v4

go 1.12

require (
	github.com/ndidplatform/migration-tools/protos_v4 v0.0.0
	github.com/tendermint/tendermint v0.32.1
)

replace github.com/ndidplatform/migration-tools/protos_v4 => ../protos_v4
