package tendermint_0_26_4

import (
	"encoding/hex"
	"log"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	dbm "github.com/tendermint/tm-db"

	"github.com/ndidplatform/migration-tools/tendermint/0_26_4/block"
	"github.com/ndidplatform/migration-tools/tendermint/0_26_4/state"
)

type tomlConfig struct {
	DBBackend string `toml:"db_backend"`
	DBPath    string `toml:"db_dir"`
}

type TendermintStateInfo struct {
	ChainID           string
	LatestBlockHeight int64
	LatestBlockHash   []byte
	LatestAppHash     []byte
}

func GetTendermintInfo(tmHome string) (tendermintStateInfo *TendermintStateInfo, err error) {
	configFile := path.Join(tmHome, "config/config.toml")
	var config tomlConfig
	if _, err = toml.DecodeFile(configFile, &config); err != nil {
		return nil, err
	}
	dbDir := path.Join(tmHome, config.DBPath)
	dbType := dbm.BackendType(config.DBBackend)
	stateDB, err := dbm.NewDB("state", dbType, dbDir)
	if err != nil {
		return nil, err
	}
	state, err := state.LoadState(stateDB)
	if err != nil {
		return nil, err
	}

	// fmt.Printf("state: %+v\n", state)

	blockDB, err := dbm.NewDB("blockstore", dbType, dbDir)
	if err != nil {
		return nil, err
	}
	blockMeta, err := block.LoadBlockMeta(blockDB, state.LastBlockHeight)
	if err != nil {
		return nil, err
	}

	// fmt.Printf("blockMeta: %+v\n", blockMeta)

	tendermintStateInfo = new(TendermintStateInfo)
	tendermintStateInfo.ChainID = blockMeta.Header.ChainID
	tendermintStateInfo.LatestBlockHeight = state.LastBlockHeight
	tendermintStateInfo.LatestBlockHash = blockMeta.BlockID.Hash
	tendermintStateInfo.LatestAppHash = blockMeta.Header.AppHash

	log.Println("===== Tendermint State Info =====")
	log.Printf("Chain ID: %s\n", tendermintStateInfo.ChainID)
	log.Printf("Latest Block Height: %d\n", tendermintStateInfo.LatestBlockHeight)
	log.Printf("Latest Block Hash: %s\n", strings.ToUpper(hex.EncodeToString(tendermintStateInfo.LatestBlockHash)))
	log.Printf("Latest App Hash: %s\n", strings.ToUpper(hex.EncodeToString(tendermintStateInfo.LatestAppHash)))

	return tendermintStateInfo, nil
}
