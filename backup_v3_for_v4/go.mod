module github.com/ndidplatform/migration-tools/backup_v3_for_v4

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/ndidplatform/migration-tools/protos_v4 v0.0.0
	github.com/tendermint/tendermint v0.32.1
)

replace github.com/ndidplatform/migration-tools/protos_v4 => ../protos_v4
