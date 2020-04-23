module github.com/ndidplatform/migration-tools/backup_v4_for_v5

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/ndidplatform/migration-tools/protos_v4 v0.0.0
	github.com/ndidplatform/migration-tools/protos_v5 v0.0.0
	github.com/tendermint/tendermint v0.32.1
)

replace github.com/ndidplatform/migration-tools/protos_v4 => ../protos_v4

replace github.com/ndidplatform/migration-tools/protos_v5 => ../protos_v5
