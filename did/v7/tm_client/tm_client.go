package tm_client

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/ndidplatform/migration-tools/log"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	readWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (readWait * 9) / 10

	// Maximum message size allowed from peer.
	// maxMessageSize = 512
)

type TmClient struct {
	logger         log.Logger
	rpcHost        string
	rpcPort        string
	connected      bool
	mustReconnect  bool
	conn           *websocket.Conn
	jsonrpcID      int
	calls          map[string]RpcCall
	callsMutex     sync.Mutex
	eventProcessWg sync.WaitGroup
	// eventNewBlockHeaderSubscription *EventNewBlockHeaderCallback
	eventNewBlockSubscribed bool
	// newBlockExpectedTxs               map[string]chan TxResult
	// newBlockExpectedTxsMutex          sync.RWMutex
	newBlockSubscriptionHandlers      []chan TxResult
	newBlockSubscriptionHandlersMutex sync.RWMutex
	send                              chan SendCall
	normalCloseChan                   chan bool
}

type SendCall struct {
	predefinedCallID string
	method           string
	params           *JsonRPCParams
	returnChan       chan RpcCallChannelReturn
}

type RpcCall struct {
	Channel chan RpcCallChannelReturn
}

type EventNewBlockHeaderCallback struct {
	Callback func(*EventDataNewBlockHeader)
}

type EventNewBlockCallback struct {
	Callback func(*EventDataNewBlock)
}

type RpcCallChannelReturn struct {
	Message []byte
	Error   *ResponseErrorJsonRPC
}

type Event struct {
	CallID  string
	Message []byte
	Error   *ResponseErrorJsonRPC
}

func New(logger log.Logger) (tmClient *TmClient, err error) {
	tmClient = &TmClient{
		logger: logger.WithFields(log.Fields{
			"module": "tm_client",
		}),
		mustReconnect: true,
	}

	tmClient.calls = make(map[string]RpcCall)
	tmClient.jsonrpcID = 1
	if err != nil {
		return nil, err
	}

	tmClient.newBlockSubscriptionHandlers = make([]chan TxResult, 0)

	tmClient.normalCloseChan = make(chan bool)

	go func() {
	OuterLoop:
		for {
			expectedClose := <-tmClient.normalCloseChan
			close(tmClient.send)
			tmClient.connected = false
			tmClient.eventNewBlockSubscribed = false
			if expectedClose {
				return
			}
			for {
				if tmClient.mustReconnect {
					time.Sleep(3 * time.Second)
					err := tmClient.reconnect()
					if err != nil {
						continue
					} else {
						continue OuterLoop
					}
				} else {
					err := tmClient.conn.Close()
					if err != nil {
						tmClient.logger.Errorf("Error closing connection: %+v", err)
					}
					return
				}
			}
		}
	}()

	return tmClient, nil
}

func (tmClient *TmClient) Connect(rpcHost string, rpcPort string) (*websocket.Conn, error) {
	tmClient.rpcHost = rpcHost
	tmClient.rpcPort = rpcPort
	u := url.URL{
		Scheme: "ws",
		Host:   tmClient.rpcHost + ":" + tmClient.rpcPort,
		Path:   "/websocket",
	}
	tmClient.logger.Infof("Connecting to Tendermint RPC websocket at %s", u.String())

	var err error
	tmClient.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	tmClient.connected = true
	tmClient.logger.Infof("Connected to Tendermint RPC websocket")

	tmClient.send = make(chan SendCall)

	go tmClient.readRoutine()
	go tmClient.writeRoutine()

	return tmClient.conn, nil
}

func (tmClient *TmClient) reconnect() error {
	u := url.URL{
		Scheme: "ws",
		Host:   tmClient.rpcHost + ":" + tmClient.rpcPort,
		Path:   "/websocket",
	}
	tmClient.logger.Infof("Reconnecting to Tendermint RPC websocket at %s", u.String())

	var err error
	tmClient.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		tmClient.logger.Errorf("Error reconnecting to Tendermint RPC websocket; err: %+v", err)
		return err
	}
	tmClient.connected = true
	tmClient.logger.Infof("Reconnected to Tendermint RPC websocket")

	tmClient.send = make(chan SendCall)

	go tmClient.readRoutine()
	go tmClient.writeRoutine()

	// Resubscribe to new block event
	for _, handler := range tmClient.newBlockSubscriptionHandlers {
		tmClient.subscribeToNewBlockEvents(handler, true)
	}

	return nil
}

func (tmClient *TmClient) Close() error {
	tmClient.eventProcessWg.Wait()
	for key := range tmClient.calls {
		close(tmClient.calls[key].Channel)
	}
	if tmClient.conn != nil {
		// Cleanly close the connection by sending a close message and then
		// waiting (with timeout) for the server to close the connection.
		err := tmClient.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			return err
		}

		tmClient.mustReconnect = false

		time.Sleep(time.Second)
		tmClient.conn.Close()
	}
	return errors.New("No connection found")
}

func (tmClient *TmClient) readRoutine() {
	// tmClient.conn.SetReadLimit(maxMessageSize)
	tmClient.conn.SetPongHandler(func(string) error {
		return tmClient.conn.SetReadDeadline(time.Now().Add(readWait))
	})
	for {
		if err := tmClient.conn.SetReadDeadline(time.Now().Add(readWait)); err != nil {
			tmClient.logger.Errorf("Failed to set read deadline: %+v", err)
		}
		_, message, err := tmClient.conn.ReadMessage()
		if err != nil {
			tmClient.logger.Debugf("Read error: %v", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				tmClient.logger.Errorf("%+v", err)
				tmClient.normalCloseChan <- false
				return
			}
			tmClient.normalCloseChan <- true
			return
		}

		tmClient.logger.Debugf("readRoutine recv: %s", message)

		var response ResponseJsonRPC
		err = json.Unmarshal(message, &response)
		if err != nil {
			tmClient.logger.Errorf("Parse response JSON error: %+v", err)
			return
		}

		tmClient.callsMutex.Lock()
		call, exist := tmClient.calls[response.ID]
		if exist {
			call.Channel <- RpcCallChannelReturn{
				message,
				response.Error,
			}
			delete(tmClient.calls, response.ID)
		}
		tmClient.callsMutex.Unlock()

		if strings.HasSuffix(response.ID, "#event") {
			event := Event{
				response.ID,
				message,
				response.Error,
			}
			tmClient.eventProcessWg.Add(1)
			go func() {
				tmClient.processEventRecv(event)
				tmClient.eventProcessWg.Done()
			}()
		}
	}
}

func (tmClient *TmClient) writeRoutine() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case sendCall, ok := <-tmClient.send:
			tmClient.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// send channel closed.
				tmClient.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			var req RequestJsonRPC

			var callID string
			if sendCall.predefinedCallID != "" {
				callID = sendCall.predefinedCallID
			} else {
				callID = strconv.Itoa(tmClient.jsonrpcID)
			}

			req.Jsonrpc = "2.0"
			req.ID = callID
			req.Method = sendCall.method
			if sendCall.params != nil {
				req.Params = *sendCall.params
			}

			tmClient.jsonrpcID = tmClient.jsonrpcID + 1

			tmClient.callsMutex.Lock()
			tmClient.calls[callID] = RpcCall{
				Channel: sendCall.returnChan,
			}
			tmClient.callsMutex.Unlock()

			tmClient.logger.Debugf("writeRoutine send: %s", req)
			err := tmClient.conn.WriteJSON(req)
			if err != nil {
				tmClient.logger.Errorf("Websocket write JSON error: %+v", err)
				return
			}
		case <-ticker.C:
			if tmClient.connected {
				tmClient.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := tmClient.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					tmClient.logger.Errorf("Websocket write message (ping) error: %+v", err)
					return
				}
			}
		}
	}
}

func (tmClient *TmClient) call(method string, params *JsonRPCParams, predefinedCallID string) ([]byte, error) {
	channel := make(chan RpcCallChannelReturn)
	sendCall := SendCall{
		predefinedCallID: predefinedCallID,
		method:           method,
		params:           params,
		returnChan:       channel,
	}
	tmClient.send <- sendCall
	channelReturn, ok := <-channel
	if !ok {
		return nil, errors.New("channel closed while call is not finished")
	}
	if channelReturn.Error != nil {
		return nil, errors.Wrap(errors.New(channelReturn.Error.Data), "JSON-RPC error")
	}
	var objmap map[string]*json.RawMessage
	err := json.Unmarshal(channelReturn.Message, &objmap)
	if err != nil {
		return nil, err
	}
	return *objmap["result"], nil
}

func (tmClient *TmClient) processEventRecv(event Event) {
	if event.Error != nil {
		tmClient.logger.Errorf("Event recv error: " + event.Error.Data)
		return
	}
	if strings.HasPrefix(event.CallID, "NewBlock") && tmClient.eventNewBlockSubscribed {
		var objmap map[string]*json.RawMessage
		err := json.Unmarshal(event.Message, &objmap)
		if err != nil {
			tmClient.logger.Errorf("%+v", err)
		}
		var newBlockEvent EventDataNewBlock
		err = json.Unmarshal(*objmap["result"], &newBlockEvent)
		if err != nil {
			tmClient.logger.Errorf("%+v", err)
		}
		tmClient.newBlockHandler(newBlockEvent)
	}
	// else if strings.HasPrefix(event.CallID, "NewBlockHeader") && tmClient.eventNewBlockHeaderSubscription != nil {
	// 	var objmap map[string]*json.RawMessage
	// 	err := json.Unmarshal(event.Message, &objmap)
	// 	if err != nil {
	// 		tmClient.logger.Error(err)
	// 	}
	// 	var newBlockHeaderEvent EventDataNewBlockHeader
	// 	err = json.Unmarshal(*objmap["result"], &newBlockHeaderEvent)
	// 	if err != nil {
	// 		tmClient.logger.Error(err)
	// 	}
	// 	tmClient.eventNewBlockHeaderSubscription.Callback(&newBlockHeaderEvent)
	// }
}

func (tmClient *TmClient) Status() (status *ResponseStatus, err error) {
	res, err := tmClient.call("status", nil, "")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(res, &status)
	if err != nil {
		return nil, err
	}
	// FIXME: delete this
	// fmt.Println(string(res))
	return status, nil
}

func (tmClient *TmClient) Block(height int) (block *ResponseBlock, err error) {
	res, err := tmClient.call("block", &JsonRPCParams{
		Height: strconv.Itoa(height),
	}, "")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(res, &block)
	if err != nil {
		return nil, err
	}
	// FIXME: delete this
	// fmt.Println(string(res))
	return block, nil
}

func (tmClient *TmClient) BlockResults(height int) (blockResults *ResponseBlockResults, err error) {
	res, err := tmClient.call("block_results", &JsonRPCParams{
		Height: strconv.Itoa(height),
	}, "")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(res, &blockResults)
	if err != nil {
		return nil, err
	}
	// FIXME: delete this
	// fmt.Println(string(res))
	return blockResults, nil
}

func (tmClient *TmClient) Query(data []byte) (queryRes *ResponseQuery, err error) {
	tmClient.logger.Debugf("query")
	res, err := tmClient.call("abci_query", &JsonRPCParams{
		Path:   base64.StdEncoding.EncodeToString([]byte("")),
		Data:   hex.EncodeToString(data),
		Height: "0",
	}, "")
	if err != nil {
		tmClient.logger.Errorf("Failed to call: %+v", err)
		return nil, err
	}
	err = json.Unmarshal(res, &queryRes)
	if err != nil {
		return nil, err
	}
	// FIXME: delete this
	// fmt.Println(string(res))
	return queryRes, nil
}

func (tmClient *TmClient) BroadcastTxCommit(tx []byte) (broadcastTxCommitResult *ResponseBroadcastTxCommit, err error) {
	tmClient.logger.Debugf("broadcast tx commit")
	res, err := tmClient.call("broadcast_tx_commit", &JsonRPCParams{
		Tx: base64.StdEncoding.EncodeToString(tx),
	}, "")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(res, &broadcastTxCommitResult)
	if err != nil {
		return nil, err
	}
	// FIXME: delete this
	// fmt.Println(string(res))
	return broadcastTxCommitResult, nil
}

func (tmClient *TmClient) BroadcastTxSync(tx []byte) (broadcastTxSyncResult *ResponseBroadcastTxSync, err error) {
	tmClient.logger.Debugf("broadcast tx sync")
	res, err := tmClient.call("broadcast_tx_sync", &JsonRPCParams{
		Tx: base64.StdEncoding.EncodeToString(tx),
	}, "")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(res, &broadcastTxSyncResult)
	if err != nil {
		return nil, err
	}
	// FIXME: delete this
	// fmt.Println(string(res))
	return broadcastTxSyncResult, nil
}

// func (tmClient *TmClient) SubscribeToNewBlockHeaderEvents(callback func(*EventDataNewBlockHeader)) (subscribeResult *ResponseSubscribe, err error) {
// 	tmClient.logger.Debugf("subscribe to new block header event")
// 	if tmClient.eventNewBlockHeaderSubscription != nil {
// 		return nil, errors.New("Already subscribed")
// 	}
// 	res, err := tmClient.call("subscribe", &JsonRPCParams{
// 		Query: "tm.event = 'NewBlockHeader'",
// 	}, "NewBlockHeader")
// 	if err != nil {
// 		return nil, err
// 	}
// 	err = json.Unmarshal(res, &subscribeResult)
// 	if err != nil {
// 		return nil, err
// 	}
// 	tmClient.eventNewBlockHeaderSubscription = &EventNewBlockHeaderCallback{
// 		Callback: callback,
// 	}
// 	// FIXME: delete this
// 	// fmt.Println(string(res))
// 	return subscribeResult, nil
// }

func (tmClient *TmClient) SubscribeToNewBlockEvents(handler chan TxResult) (subscribeResult *ResponseSubscribe, err error) {
	return tmClient.subscribeToNewBlockEvents(handler, false)
}

func (tmClient *TmClient) subscribeToNewBlockEvents(handler chan TxResult, subscribeOnReconnect bool) (subscribeResult *ResponseSubscribe, err error) {
	tmClient.logger.Debugf("subscribe to new block event")
	if tmClient.eventNewBlockSubscribed {
		return nil, errors.New("Already subscribed")
	}
	res, err := tmClient.call("subscribe", &JsonRPCParams{
		Query: "tm.event = 'NewBlock'",
	}, "NewBlock#event")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(res, &subscribeResult)
	if err != nil {
		return nil, err
	}
	// tmClient.newBlockExpectedTxs = make(map[string]chan TxResult)
	tmClient.eventNewBlockSubscribed = true

	if handler != nil && !subscribeOnReconnect {
		tmClient.newBlockSubscriptionHandlersMutex.Lock()
		tmClient.newBlockSubscriptionHandlers = append(tmClient.newBlockSubscriptionHandlers, handler)
		tmClient.newBlockSubscriptionHandlersMutex.Unlock()
	}

	return subscribeResult, nil
}
