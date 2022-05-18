package tm_client

import "time"

type JsonRPC struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
}

type RequestJsonRPC struct {
	JsonRPC
	Method string        `json:"method"`
	Params JsonRPCParams `json:"params"`
}

type JsonRPCParams struct {
	Tx     string `json:"tx"`
	Data   string `json:"data"`
	Path   string `json:"path"`
	Query  string `json:"query"`
	Hash   string `json:"hash"`
	Height string `json:"height"`
}

type ResponseJsonRPC struct {
	JsonRPC
	Error *ResponseErrorJsonRPC `json:"error"`
}

type ResponseErrorJsonRPC struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type ResponseStatus struct {
	NodeInfo struct {
		ProtocolVersion struct {
			P2P   string `json:"p2p"`
			Block string `json:"block"`
			App   string `json:"app"`
		} `json:"protocol_version"`
		ID         string `json:"id"`
		ListenAddr string `json:"listen_addr"`
		Network    string `json:"network"`
		Version    string `json:"version"`
		Channels   string `json:"channels"`
		Moniker    string `json:"moniker"`
		Other      struct {
			TxIndex    string `json:"tx_index"`
			RPCAddress string `json:"rpc_address"`
		} `json:"other"`
	} `json:"node_info"`
	SyncInfo struct {
		LatestBlockHash   string    `json:"latest_block_hash"`
		LatestAppHash     string    `json:"latest_app_hash"`
		LatestBlockHeight string    `json:"latest_block_height"`
		LatestBlockTime   time.Time `json:"latest_block_time"`
		CatchingUp        bool      `json:"catching_up"`
	} `json:"sync_info"`
	ValidatorInfo struct {
		Address string `json:"address"`
		PubKey  struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"pub_key"`
		VotingPower string `json:"voting_power"`
	} `json:"validator_info"`
}

type ResponseBlock struct {
	BlockMeta struct {
		BlockID struct {
			Hash  string `json:"hash"`
			Parts struct {
				Total int    `json:"total"`
				Hash  string `json:"hash"`
			} `json:"parts"`
		} `json:"block_id"`
		Header struct {
			Version struct {
				Block string `json:"block"`
				App   string `json:"app"`
			} `json:"version"`
			ChainID     string    `json:"chain_id"`
			Height      string    `json:"height"`
			Time        time.Time `json:"time"`
			NumTxs      string    `json:"num_txs"`
			TotalTxs    string    `json:"total_txs"`
			LastBlockID struct {
				Hash  string `json:"hash"`
				Parts struct {
					Total int    `json:"total"`
					Hash  string `json:"hash"`
				} `json:"parts"`
			} `json:"last_block_id"`
			LastCommitHash     string `json:"last_commit_hash"`
			DataHash           string `json:"data_hash"`
			ValidatorsHash     string `json:"validators_hash"`
			NextValidatorsHash string `json:"next_validators_hash"`
			ConsensusHash      string `json:"consensus_hash"`
			AppHash            string `json:"app_hash"`
			LastResultsHash    string `json:"last_results_hash"`
			EvidenceHash       string `json:"evidence_hash"`
			ProposerAddress    string `json:"proposer_address"`
		} `json:"header"`
	} `json:"block_meta"`
	Block struct {
		Header struct {
			Version struct {
				Block string `json:"block"`
				App   string `json:"app"`
			} `json:"version"`
			ChainID     string    `json:"chain_id"`
			Height      string    `json:"height"`
			Time        time.Time `json:"time"`
			NumTxs      string    `json:"num_txs"`
			TotalTxs    string    `json:"total_txs"`
			LastBlockID struct {
				Hash  string `json:"hash"`
				Parts struct {
					Total int    `json:"total"`
					Hash  string `json:"hash"`
				} `json:"parts"`
			} `json:"last_block_id"`
			LastCommitHash     string `json:"last_commit_hash"`
			DataHash           string `json:"data_hash"`
			ValidatorsHash     string `json:"validators_hash"`
			NextValidatorsHash string `json:"next_validators_hash"`
			ConsensusHash      string `json:"consensus_hash"`
			AppHash            string `json:"app_hash"`
			LastResultsHash    string `json:"last_results_hash"`
			EvidenceHash       string `json:"evidence_hash"`
			ProposerAddress    string `json:"proposer_address"`
		} `json:"header"`
		Data struct {
			Txs []string `json:"txs"`
		} `json:"data"`
		Evidence struct {
			Evidence interface{} `json:"evidence"`
		} `json:"evidence"`
		LastCommit struct {
			BlockID struct {
				Hash  string `json:"hash"`
				Parts struct {
					Total int    `json:"total"`
					Hash  string `json:"hash"`
				} `json:"parts"`
			} `json:"block_id"`
			Precommits []struct {
				Type    int    `json:"type"`
				Height  string `json:"height"`
				Round   string `json:"round"`
				BlockID struct {
					Hash  string `json:"hash"`
					Parts struct {
						Total int    `json:"total"`
						Hash  string `json:"hash"`
					} `json:"parts"`
				} `json:"block_id"`
				Timestamp        time.Time `json:"timestamp"`
				ValidatorAddress string    `json:"validator_address"`
				ValidatorIndex   string    `json:"validator_index"`
				Signature        string    `json:"signature"`
			} `json:"precommits"`
		} `json:"last_commit"`
	} `json:"block"`
}

type ResponseBlockResults struct {
	Height     string `json:"height"`
	TxsResults []struct {
		Code      uint32 `json:"code"`
		Data      string `json:"data"`
		Log       string `json:"log"`
		Info      string `json:"info"`
		GasWanted string `json:"gasWanted"`
		GasUsed   string `json:"gasUsed"`
		Events    []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"events"`
		Codespace string `json:"codespace"`
	} `json:"txs_results"`
	BeginBlockEvents []struct {
		Type       string `json:"type"`
		Attributes []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"attributes"`
	} `json:"begin_block_events"`
	EndBlock []struct {
		Type       string `json:"type"`
		Attributes []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"attributes"`
	} `json:"end_block"`
	ValidatorUpdates []struct {
		PubKey struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"pub_key"`
		Power string `json:"power"`
	} `json:"validator_updates"`
	ConsensusParamUpdates struct {
		Block struct {
			MaxBytes   string `json:"max_bytes"`
			MaxGas     string `json:"max_gas"`
			TimeIotaMs string `json:"time_iota_ms"`
		} `json:"block"`
		Evidence struct {
			MaxAge string `json:"max_age"`
		} `json:"evidence"`
		Validator struct {
			PubKeyTypes []string `json:"pub_key_types"`
		} `json:"validator"`
	} `json:"consensus_param_updates"`
}

type ResponseBroadcastTxCommit struct {
	CheckTx struct {
		Code      int           `json:"code"`
		Data      interface{}   `json:"data"`
		Log       string        `json:"log"`
		Info      string        `json:"info"`
		GasWanted string        `json:"gasWanted"`
		GasUsed   string        `json:"gasUsed"`
		Events    []interface{} `json:"events"`
		Codespace string        `json:"codespace"`
	} `json:"check_tx"`
	DeliverTx struct {
		Code      int    `json:"code"`
		Data      string `json:"data"`
		Log       string `json:"log"`
		Info      string `json:"info"`
		GasWanted string `json:"gasWanted"`
		GasUsed   string `json:"gasUsed"`
		Events    []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"events"`
		Codespace string `json:"codespace"`
	} `json:"deliver_tx"`
	Hash   string `json:"hash"`
	Height string `json:"height"`
}

type ResponseQuery struct {
	Response struct {
		Log    string `json:"log"`
		Value  string `json:"value"`
		Height string `json:"height"`
	} `json:"response"`
}

type ResponseBroadcastTxSync struct {
	Code int    `json:"code"`
	Data string `json:"data"`
	Log  string `json:"log"`
	Hash string `json:"hash"`
}

type ResponseSubscribe struct {
}

type BlockHeader struct {
	Version struct {
		Block string `json:"block"`
		App   string `json:"app"`
	} `json:"version"`
	ChainID     string    `json:"chain_id"`
	Height      string    `json:"height"`
	Time        time.Time `json:"time"`
	NumTxs      string    `json:"num_txs"`
	TotalTxs    string    `json:"total_txs"`
	LastBlockID struct {
		Hash  string `json:"hash"`
		Parts struct {
			Total int    `json:"total"`
			Hash  string `json:"hash"`
		} `json:"parts"`
	} `json:"last_block_id"`
	LastCommitHash     string `json:"last_commit_hash"`
	DataHash           string `json:"data_hash"`
	ValidatorsHash     string `json:"validators_hash"`
	NextValidatorsHash string `json:"next_validators_hash"`
	ConsensusHash      string `json:"consensus_hash"`
	AppHash            string `json:"app_hash"`
	LastResultsHash    string `json:"last_results_hash"`
	EvidenceHash       string `json:"evidence_hash"`
	ProposerAddress    string `json:"proposer_address"`
}

type EventDataNewBlockHeader struct {
	Query string `json:"query"`
	Data  struct {
		Type  string `json:"type"`
		Value struct {
			Header           BlockHeader `json:"header"`
			ResultBeginBlock struct {
			} `json:"result_begin_block"`
			ResultEndBlock struct {
				ValidatorUpdates interface{} `json:"validator_updates"`
			} `json:"result_end_block"`
		} `json:"value"`
	} `json:"data"`
	Events struct {
		TmEvent []string `json:"tm.event"`
	} `json:"events"`
}

type EventDataNewBlock struct {
	Query string `json:"query"`
	Data  struct {
		Type  string `json:"type"`
		Value struct {
			Block struct {
				Header BlockHeader `json:"header"`
				Data   struct {
					Txs []string `json:"txs"`
				} `json:"data"`
				Evidence struct {
					Evidence interface{} `json:"evidence"`
				} `json:"evidence"`
				LastCommit struct {
					BlockID struct {
						Hash  string `json:"hash"`
						Parts struct {
							Total int    `json:"total"`
							Hash  string `json:"hash"`
						} `json:"parts"`
					} `json:"block_id"`
					Precommits []struct {
						Type    int    `json:"type"`
						Height  string `json:"height"`
						Round   string `json:"round"`
						BlockID struct {
							Hash  string `json:"hash"`
							Parts struct {
								Total int    `json:"total"`
								Hash  string `json:"hash"`
							} `json:"parts"`
						} `json:"block_id"`
						Timestamp        time.Time `json:"timestamp"`
						ValidatorAddress string    `json:"validator_address"`
						ValidatorIndex   string    `json:"validator_index"`
						Signature        string    `json:"signature"`
					} `json:"precommits"`
				} `json:"last_commit"`
			} `json:"block"`
			ResultBeginBlock struct {
			} `json:"result_begin_block"`
			ResultEndBlock struct {
				ValidatorUpdates interface{} `json:"validator_updates"`
			} `json:"result_end_block"`
		} `json:"value"`
	} `json:"data"`
	Events struct {
		TmEvent []string `json:"tm.event"`
	} `json:"events"`
}
