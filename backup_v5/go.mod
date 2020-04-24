module github.com/ndidplatform/migration-tools/backup_v5

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/ndidplatform/migration-tools/protos_v5 v0.0.0
	github.com/tendermint/tendermint v0.33.3
	github.com/tendermint/tm-db v0.4.1
)

replace github.com/ndidplatform/migration-tools/protos_v5 => ../protos_v5
