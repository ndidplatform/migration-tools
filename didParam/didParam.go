package didParam

type SetInitDataParam struct {
	KVList []KeyValue `json:"kv_list"`
}

type KeyValue struct {
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

type InitNDIDParam struct {
	NodeID           string `json:"node_id"`
	PublicKey        string `json:"public_key"`
	MasterPublicKey  string `json:"master_public_key"`
	ChainHistoryInfo string `json:"chain_history_info"`
}

type EndInitParam struct{}
