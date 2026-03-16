package tm_client

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
)

type TxResult struct {
	Tx              []byte
	TxHashHex       string
	DeliverTxResult *deliverTxResult
}

type deliverTxResult struct {
	Data string
	Log  string
}

type AddExpectedTxParams struct {
	TxHashHex  string
	ResultChan chan TxResult
}

// func (tmClient *TmClient) addNewBlockExpectedTx(txHashHex string, resultChan chan TxResult) {
// 	tmClient.newBlockExpectedTxsMutex.Lock()
// 	defer tmClient.newBlockExpectedTxsMutex.Unlock()
// 	tmClient.newBlockExpectedTxs[txHashHex] = resultChan
// }

func (tmClient *TmClient) newBlockHandler(event EventDataNewBlock) {
	// Get block result when get new block event
	blockHeight, err := strconv.Atoi(event.Data.Value.Block.Header.Height)
	if err != nil {
		tmClient.logger.Errorf("err: %+v", err)
	}
	blockResult, err := tmClient.BlockResults(blockHeight)
	if err != nil {
		tmClient.logger.Errorf("get block results err: %+v", err)
	}

	tmClient.logger.Infof("Handle TM new block height: %d", blockHeight)

	// Get the result & Watch wallet ID that's associated with this node

	for txIndex, tx := range event.Data.Value.Block.Data.Txs {
		tx, err := base64.StdEncoding.DecodeString(tx)
		if err != nil {
			tmClient.logger.Errorf("error decoding tx string from new block event: %+v", err)
		}
		txHash := sha256.Sum256([]byte(tx))
		txHashHex := hex.EncodeToString(txHash[:])

		result := blockResult.TxsResults[txIndex]

		// tmClient.newBlockExpectedTxsMutex.RLock()
		// resultChan, ok := tmClient.newBlockExpectedTxs[txHashHex]
		// tmClient.newBlockExpectedTxsMutex.RUnlock()
		// if ok {
		// 	tmClient.logger.Infof("Expected Tx hash: %s", txHashHex)
		// 	tmClient.logger.Infof("Block height: %d", event.Data.Value.Block.Header.Height)

		// 	tmClient.logger.Infof("Tx: ", txHashHex, "Data: ", result.Data, "Log: ", result.Log)
		// 	resultChan <- TxResult{
		// 		Tx: tx,
		// 		DeliverTxResult: &deliverTxResult{
		// 			Data: result.Data,
		// 			Log:  result.Log,
		// 		},
		// 	}
		// 	tmClient.newBlockExpectedTxsMutex.Lock()
		// 	delete(tmClient.newBlockExpectedTxs, txHashHex)
		// 	tmClient.newBlockExpectedTxsMutex.Unlock()
		// }

		tmClient.newBlockSubscriptionHandlersMutex.RLock()
		for _, handler := range tmClient.newBlockSubscriptionHandlers {
			handler <- TxResult{
				Tx:        tx,
				TxHashHex: txHashHex,
				DeliverTxResult: &deliverTxResult{
					Data: result.Data,
					Log:  result.Log,
				},
			}
		}
		tmClient.newBlockSubscriptionHandlersMutex.RUnlock()
	}
}
