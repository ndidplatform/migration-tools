/**
 * Copyright (c) 2018, 2019 National Digital ID COMPANY LIMITED
 *
 * This file is part of NDID software.
 *
 * NDID is the free software: you can redistribute it and/or modify it under
 * the terms of the Affero GNU General Public License as published by the
 * Free Software Foundation, either version 3 of the License, or any later
 * version.
 *
 * NDID is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 * See the Affero GNU General Public License for more details.
 *
 * You should have received a copy of the Affero GNU General Public License
 * along with the NDID source code. If not, see https://www.gnu.org/licenses/agpl.txt.
 *
 * Please contact info@ndid.co.th for any further questions
 *
 */

package v7

import (
	"bufio"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"

	protoParam "github.com/ndidplatform/migration-tools/did/v7/protos/param"
	protoTm "github.com/ndidplatform/migration-tools/did/v7/protos/tendermint"
	"github.com/ndidplatform/migration-tools/did/v7/tm_client"
	"github.com/ndidplatform/migration-tools/proto"
	tmRand "github.com/ndidplatform/migration-tools/rand"

	_log "github.com/ndidplatform/migration-tools/log"
	"github.com/ndidplatform/migration-tools/utils"
)

var currentChainID string

func Restore(
	ndidID string,
	backupDataDir string,
	backupDataFileName string,
	chainHistoryFileName string,
	keyDir string,
	tendermintRPCHost string,
	tendermintRPCPort string,
) (err error) {
	ndidKeyFile, err := os.Open(keyDir + "ndid")
	if err != nil {
		return err
	}
	defer ndidKeyFile.Close()
	data, err := ioutil.ReadAll(ndidKeyFile)
	if err != nil {
		return err
	}
	ndidMasterKeyFile, err := os.Open(keyDir + "ndid_master")
	if err != nil {
		return err
	}
	defer ndidMasterKeyFile.Close()

	logger, err := _log.NewLogger(&_log.Configuration{
		EnableConsole:     true,
		ConsoleLevel:      "info",
		ConsoleJSONFormat: false,
		Color:             true,
	}, _log.InstanceGoLogger)
	if err != nil {
		return err
	}
	tmClient, err := tm_client.New(logger)
	if err != nil {
		return err
	}
	_, err = tmClient.Connect(tendermintRPCHost, tendermintRPCPort)
	if err != nil {
		return err
	}
	defer tmClient.Close()

	tmStatus, err := tmClient.Status()
	if err != nil {
		return err
	}
	currentChainID = tmStatus.NodeInfo.Network

	txResultChan := make(chan tm_client.TxResult)
	tmClient.SubscribeToNewBlockEvents(txResultChan)

	var deliverTxLogChanMap map[string]chan string = make(map[string]chan string)
	var deliverTxLogChanMutex sync.RWMutex

	go func() {
		for {
			txResult, ok := <-txResultChan
			if !ok {
				return
			}
			deliverTxLogChanMutex.RLock()
			deliverTxChan, ok := deliverTxLogChanMap[txResult.TxHashHex]
			deliverTxLogChanMutex.RUnlock()
			if ok {
				deliverTxChan <- txResult.DeliverTxResult.Log
			}
		}
	}()

	dataMaster, err := ioutil.ReadAll(ndidMasterKeyFile)
	if err != nil {
		return err
	}
	ndidPrivKey := utils.GetPrivateKeyFromString(string(data))
	ndidMasterPrivKey := utils.GetPrivateKeyFromString(string(dataMaster))
	err = initNDID(
		tmClient,
		ndidPrivKey,
		ndidMasterPrivKey,
		ndidID,
		backupDataDir,
		chainHistoryFileName,
	)
	if err != nil {
		return err
	}
	file, err := os.Open(backupDataDir + backupDataFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	estimatedTxSizeBytes := 300000
	size := 0
	count := 0
	nTx := 0

	maxWorkerCount := 3000
	sem := make(chan struct{}, maxWorkerCount)

	var wg sync.WaitGroup

	worker := func(
		param SetInitDataParam,
		ndidKey *rsa.PrivateKey,
		ndidID string,
		nTx int,
	) {
		defer wg.Done()
		txHashHex, err := setInitData_pb(tmClient, param, ndidPrivKey, ndidID)
		if err != nil {
			panic(err)
		}
		deliverTxLogChan := make(chan string)

		deliverTxLogChanMutex.Lock()
		deliverTxLogChanMap[txHashHex] = deliverTxLogChan
		deliverTxLogChanMutex.Unlock()

		deliverTxLog := <-deliverTxLogChan
		log.Printf("SetInitData (kv count: %d) DeliverTx log: %s\n", nTx, deliverTxLog)

		if deliverTxLog != "success" {
			log.Fatalf("SetInitData (kv count: %d) DeliverTx failed: %s\n", nTx, deliverTxLog)
			panic(fmt.Errorf("err"))
		}
		<-sem
	}

	var param SetInitDataParam
	param.KVList = make([]KeyValue, 0)
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Fatalf("read err: %+v\n", err)
				return err
			}
		}

		var kv KeyValue
		err = json.Unmarshal([]byte(line), &kv)
		if err != nil {
			panic(err)
		}
		param.KVList = append(param.KVList, kv)
		count++
		size += len(kv.Key) + len(kv.Value)
		nTx++
		if size > estimatedTxSizeBytes {
			sem <- struct{}{}
			wg.Add(1)
			go worker(param, ndidPrivKey, ndidID, nTx)

			log.Printf("Number of kv in param: %d\n", count)
			log.Printf("Total number of kv: %d\n", nTx)
			count = 0
			size = 0
			param.KVList = make([]KeyValue, 0)
		}
	}
	if count > 0 {
		sem <- struct{}{}
		wg.Add(1)
		go worker(param, ndidPrivKey, ndidID, nTx)

		log.Printf("Number of kv in param: %d\n", count)
		log.Printf("Total number of kv: %d\n", nTx)
	}

	wg.Wait()

	err = endInit(tmClient, ndidPrivKey, ndidID)
	if err != nil {
		return err
	}

	return nil
}

func SetNodeKeys(
	ndidID string,
	nodeMasterPublicKeyFilepath string,
	nodePublicKeyFilepath string,
	keyDir string,
	tendermintRPCHost string,
	tendermintRPCPort string,
) (err error) {
	ndidMasterKeyFile, err := os.Open(keyDir + "ndid_master")
	if err != nil {
		return err
	}
	defer ndidMasterKeyFile.Close()
	dataMaster, err := ioutil.ReadAll(ndidMasterKeyFile)
	if err != nil {
		return err
	}

	// new public keys
	ndidPublicKeyFile, err := os.Open(nodePublicKeyFilepath)
	if err != nil {
		return err
	}
	defer ndidPublicKeyFile.Close()
	ndidNodePublicKey, err := ioutil.ReadAll(ndidPublicKeyFile)
	if err != nil {
		return err
	}
	ndidMasterPublicKeyFile, err := os.Open(nodeMasterPublicKeyFilepath)
	if err != nil {
		return err
	}
	defer ndidMasterPublicKeyFile.Close()
	ndidNodeMasterPublicKey, err := ioutil.ReadAll(ndidMasterPublicKeyFile)
	if err != nil {
		return err
	}

	logger, err := _log.NewLogger(&_log.Configuration{
		EnableConsole:     true,
		ConsoleLevel:      "info",
		ConsoleJSONFormat: false,
		Color:             true,
	}, _log.InstanceGoLogger)
	if err != nil {
		return err
	}
	tmClient, err := tm_client.New(logger)
	if err != nil {
		return err
	}
	_, err = tmClient.Connect(tendermintRPCHost, tendermintRPCPort)
	if err != nil {
		return err
	}
	defer tmClient.Close()

	ndidMasterPrivKey := utils.GetPrivateKeyFromString(string(dataMaster))
	err = updateNode(
		tmClient,
		ndidMasterPrivKey,
		string(ndidNodePublicKey),
		string(ndidNodeMasterPublicKey),
		ndidID,
	)
	if err != nil {
		return err
	}

	return nil
}

func initNDID(
	tmClient *tm_client.TmClient,
	ndidKey *rsa.PrivateKey,
	ndidMasterKey *rsa.PrivateKey,
	ndidID string,
	backupDataDir string,
	chainHistoryFileName string,
) (err error) {
	chainHistoryData, err := ioutil.ReadFile(backupDataDir + chainHistoryFileName)
	if err != nil {
		return err
	}
	ndidPublicKeyBytes, err := utils.GeneratePublicKey(&ndidKey.PublicKey)
	if err != nil {
		return err
	}
	ndidMasterPublicKeyBytes, err := utils.GeneratePublicKey(&ndidMasterKey.PublicKey)
	if err != nil {
		return err
	}
	var initNDIDparam InitNDIDParam
	initNDIDparam.NodeID = ndidID
	initNDIDparam.PublicKey = string(ndidPublicKeyBytes)
	initNDIDparam.MasterPublicKey = string(ndidMasterPublicKeyBytes)
	initNDIDparam.ChainHistoryInfo = string(chainHistoryData)
	paramJSON, err := json.Marshal(initNDIDparam)
	if err != nil {
		return err
	}
	fnName := "InitNDID"
	nonce := tmRand.Bytes(32)
	tempPSSmessage := append([]byte(fnName), paramJSON...)
	tempPSSmessage = append(tempPSSmessage, []byte(currentChainID)...)
	tempPSSmessage = append(tempPSSmessage, nonce...)
	PSSmessage := []byte(base64.StdEncoding.EncodeToString(tempPSSmessage))
	newhash := crypto.SHA256
	pssh := newhash.New()
	pssh.Write(PSSmessage)
	hashed := pssh.Sum(nil)
	signature, err := rsa.SignPKCS1v15(rand.Reader, ndidKey, newhash, hashed)
	if err != nil {
		return err
	}

	var tx protoTm.Tx
	tx.Method = string(fnName)
	tx.Params = paramJSON
	tx.ChainId = currentChainID
	tx.Nonce = nonce
	tx.Signature = signature
	tx.NodeId = string(initNDIDparam.NodeID)

	txByte, err := proto.Marshal(&tx)
	if err != nil {
		return err
	}

	// result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(initNDIDparam.NodeID))
	result, err := tmClient.BroadcastTxCommit(txByte)
	if err != nil {
		return err
	}

	log.Println("InitNDID DeliverTx log:", result.DeliverTx.Log)

	return nil
}

func setInitData(
	tmClient *tm_client.TmClient,
	param SetInitDataParam,
	ndidKey *rsa.PrivateKey,
	ndidID string,
) (txHashHex string, err error) {
	paramJSON, err := json.Marshal(param)
	if err != nil {
		return "", err
	}
	fnName := "SetInitData"
	// nonce := base64.StdEncoding.EncodeToString([]byte(tmRand.Str(12)))
	// tempPSSmessage := append([]byte(fnName), paramJSON...)
	// tempPSSmessage = append(tempPSSmessage, []byte(nonce)...)
	// PSSmessage := []byte(base64.StdEncoding.EncodeToString(tempPSSmessage))
	// newhash := crypto.SHA256
	// pssh := newhash.New()
	// pssh.Write(PSSmessage)
	// hashed := pssh.Sum(nil)
	// signature, err := rsa.SignPKCS1v15(rand.Reader, ndidKey, newhash, hashed)
	// if err != nil {
	// 	return "", err
	// }

	var tx protoTm.Tx
	tx.Method = string(fnName)
	tx.Params = paramJSON
	// tx.Nonce = []byte(nonce)
	// tx.Signature = signature
	tx.NodeId = ndidID

	txByte, err := proto.Marshal(&tx)
	if err != nil {
		return "", err
	}

	txHash := sha256.Sum256([]byte(txByte))
	txHashHex = hex.EncodeToString(txHash[:])

	// result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(ndidID))
	// result, err := tmClient.BroadcastTxCommit(txByte)
	result, err := tmClient.BroadcastTxSync(txByte)
	if err != nil {
		return "", err
	}
	log.Printf("SetInitData CheckTx code: %d log: %s\n", result.Code, result.Log)
	// log.Printf("SetInitData DeliverTx log: %s\n", result.DeliverTx.Log)

	if result.Code != 0 {
		return "", fmt.Errorf("SetInitData CheckTx non-0 code: %d", result.Code)
	}

	// if result.DeliverTx.Log != "success" {
	// 	// pause 3 sec and retry again
	// 	log.Printf("Retry...\n")
	// 	time.Sleep(3 * time.Second)
	// 	txHashHex, err = setInitData(tmClient, param, ndidKey, ndidID)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }

	return txHashHex, nil
}

func setInitData_pb(
	tmClient *tm_client.TmClient,
	param SetInitDataParam,
	ndidKey *rsa.PrivateKey,
	ndidID string,
) (txHashHex string, err error) {
	var paramPb protoParam.SetInitDataParam
	paramPb.KvList = make([]*protoParam.KeyValue, 0)
	for _, kv := range param.KVList {
		paramPb.KvList = append(paramPb.KvList, &protoParam.KeyValue{
			Key:   kv.Key,
			Value: kv.Value,
		})
	}

	paramPbByte, err := proto.Marshal(&paramPb)
	if err != nil {
		return "", err
	}

	fnName := "SetInitData_pb"
	// nonce := base64.StdEncoding.EncodeToString([]byte(tmRand.Str(12)))
	// tempPSSmessage := append([]byte(fnName), paramPbByte...)
	// tempPSSmessage = append(tempPSSmessage, []byte(nonce)...)
	// PSSmessage := []byte(base64.StdEncoding.EncodeToString(tempPSSmessage))
	// newhash := crypto.SHA256
	// pssh := newhash.New()
	// pssh.Write(PSSmessage)
	// hashed := pssh.Sum(nil)
	// signature, err := rsa.SignPKCS1v15(rand.Reader, ndidKey, newhash, hashed)
	// if err != nil {
	// 	return "", err
	// }

	var tx protoTm.Tx
	tx.Method = string(fnName)
	tx.Params = paramPbByte
	// tx.Nonce = []byte(nonce)
	// tx.Signature = signature
	tx.NodeId = ndidID

	txByte, err := proto.Marshal(&tx)
	if err != nil {
		return "", err
	}

	txHash := sha256.Sum256([]byte(txByte))
	txHashHex = hex.EncodeToString(txHash[:])

	// result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(ndidID))
	// result, err := tmClient.BroadcastTxCommit(txByte)
	result, err := tmClient.BroadcastTxSync(txByte)
	if err != nil {
		return "", err
	}
	log.Printf("SetInitData_pb CheckTx code: %d log: %s\n", result.Code, result.Log)
	// log.Printf("SetInitData DeliverTx log: %s\n", result.DeliverTx.Log)

	if result.Code != 0 {
		return "", fmt.Errorf("SetInitData_pb CheckTx non-0 code: %d", result.Code)
	}

	// if result.DeliverTx.Log != "success" {
	// 	// pause 3 sec and retry again
	// 	log.Printf("Retry...\n")
	// 	time.Sleep(3 * time.Second)
	// 	txHashHex, err = setInitData(tmClient, param, ndidKey, ndidID)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }

	return txHashHex, nil
}

func endInit(
	tmClient *tm_client.TmClient,
	ndidKey *rsa.PrivateKey,
	ndidID string,
) (err error) {
	var param EndInitParam
	paramJSON, err := json.Marshal(param)
	if err != nil {
		return err
	}
	fnName := "EndInit"
	nonce := tmRand.Bytes(32)
	tempPSSmessage := append([]byte(fnName), paramJSON...)
	tempPSSmessage = append(tempPSSmessage, []byte(currentChainID)...)
	tempPSSmessage = append(tempPSSmessage, nonce...)
	PSSmessage := []byte(base64.StdEncoding.EncodeToString(tempPSSmessage))
	newhash := crypto.SHA256
	pssh := newhash.New()
	pssh.Write(PSSmessage)
	hashed := pssh.Sum(nil)
	signature, err := rsa.SignPKCS1v15(rand.Reader, ndidKey, newhash, hashed)
	if err != nil {
		return err
	}

	var tx protoTm.Tx
	tx.Method = string(fnName)
	tx.Params = paramJSON
	tx.ChainId = currentChainID
	tx.Nonce = nonce
	tx.Signature = signature
	tx.NodeId = ndidID

	txByte, err := proto.Marshal(&tx)
	if err != nil {
		return err
	}

	// result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(nodeID))
	result, err := tmClient.BroadcastTxCommit(txByte)
	if err != nil {
		return err
	}
	log.Println("EndInit DeliverTx log:", result.DeliverTx.Log)

	return nil
}

func updateNode(
	tmClient *tm_client.TmClient,
	ndidKey *rsa.PrivateKey,
	ndidPublicKeyPem string,
	ndidMasterPublicKeyPem string,
	ndidID string,
) (err error) {
	var updateNodeParam UpdateNodeParam
	updateNodeParam.PublicKey = ndidPublicKeyPem
	updateNodeParam.MasterPublicKey = ndidMasterPublicKeyPem
	paramJSON, err := json.Marshal(updateNodeParam)
	if err != nil {
		return err
	}
	fnName := "UpdateNode"
	nonce := tmRand.Bytes(32)
	tempPSSmessage := append([]byte(fnName), paramJSON...)
	tempPSSmessage = append(tempPSSmessage, []byte(currentChainID)...)
	tempPSSmessage = append(tempPSSmessage, nonce...)
	PSSmessage := []byte(base64.StdEncoding.EncodeToString(tempPSSmessage))
	newhash := crypto.SHA256
	pssh := newhash.New()
	pssh.Write(PSSmessage)
	hashed := pssh.Sum(nil)
	signature, err := rsa.SignPKCS1v15(rand.Reader, ndidKey, newhash, hashed)
	if err != nil {
		return err
	}

	var tx protoTm.Tx
	tx.Method = string(fnName)
	tx.Params = paramJSON
	tx.ChainId = currentChainID
	tx.Nonce = []byte(nonce)
	tx.Signature = signature
	tx.NodeId = ndidID

	txByte, err := proto.Marshal(&tx)
	if err != nil {
		return err
	}

	// result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(initNDIDparam.NodeID))
	result, err := tmClient.BroadcastTxCommit(txByte)
	if err != nil {
		return err
	}

	log.Printf("UpdateNode CheckTx code: %d log: %s\n", result.CheckTx.Code, result.CheckTx.Log)
	log.Printf("UpdateNode DeliverTx code: %d log: %s\n", result.DeliverTx.Code, result.DeliverTx.Log)

	if result.CheckTx.Code != 0 {
		return fmt.Errorf("UpdateNode CheckTx non-0 code: %d", result.CheckTx.Code)
	}

	if result.DeliverTx.Code != 0 {
		return fmt.Errorf("UpdateNode DeliverTx non-0 code: %d", result.DeliverTx.Code)
	}

	return nil
}
