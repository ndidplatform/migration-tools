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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	protoTm "github.com/ndidplatform/migration-tools/did/v7/protos/tendermint"
	"github.com/ndidplatform/migration-tools/did/v7/tm_client"
	"github.com/ndidplatform/migration-tools/proto"
	tmRand "github.com/ndidplatform/migration-tools/rand"

	_log "github.com/ndidplatform/migration-tools/log"
	"github.com/ndidplatform/migration-tools/utils"
)

var tendermintRPCAddress string

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

	tendermintRPCAddress = fmt.Sprintf("http://%s:%s", tendermintRPCHost, tendermintRPCPort)

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

	var wg sync.WaitGroup

	var param SetInitDataParam
	param.KVList = make([]KeyValue, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		jsonStr := scanner.Text()
		var kv KeyValue
		err := json.Unmarshal([]byte(jsonStr), &kv)
		if err != nil {
			panic(err)
		}
		param.KVList = append(param.KVList, kv)
		count++
		size += len(kv.Key) + len(kv.Value)
		nTx++
		if size > estimatedTxSizeBytes {
			wg.Add(1)
			go func(tmClient *tm_client.TmClient,
				param SetInitDataParam,
				ndidKey *rsa.PrivateKey,
				ndidID string) {
				defer wg.Done()
				err = setInitData(tmClient, param, ndidPrivKey, ndidID)
				if err != nil {
					panic(err)
				}
			}(tmClient, param, ndidPrivKey, ndidID)

			log.Printf("Number of kv in param: %d\n", count)
			log.Printf("Total number of kv: %d\n", nTx)
			count = 0
			size = 0
			param.KVList = make([]KeyValue, 0)
		}
	}
	if count > 0 {
		wg.Add(1)
		go func(tmClient *tm_client.TmClient,
			param SetInitDataParam,
			ndidKey *rsa.PrivateKey,
			ndidID string) {
			defer wg.Done()
			err = setInitData(tmClient, param, ndidPrivKey, ndidID)
			if err != nil {
				panic(err)
			}
		}(tmClient, param, ndidPrivKey, ndidID)
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
	nonce := base64.StdEncoding.EncodeToString([]byte(tmRand.Str(12)))
	tempPSSmessage := append([]byte(fnName), paramJSON...)
	tempPSSmessage = append(tempPSSmessage, []byte(nonce)...)
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
	tx.Params = string(paramJSON)
	tx.Nonce = []byte(nonce)
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
) (err error) {
	paramJSON, err := json.Marshal(param)
	if err != nil {
		return err
	}
	fnName := "SetInitData"
	nonce := base64.StdEncoding.EncodeToString([]byte(tmRand.Str(12)))
	tempPSSmessage := append([]byte(fnName), paramJSON...)
	tempPSSmessage = append(tempPSSmessage, []byte(nonce)...)
	PSSmessage := []byte(base64.StdEncoding.EncodeToString(tempPSSmessage))
	newhash := crypto.SHA256
	pssh := newhash.New()
	pssh.Write(PSSmessage)
	hashed := pssh.Sum(nil)
	signature, err := rsa.SignPKCS1v15(rand.Reader, ndidKey, newhash, hashed)
	if err != nil {
		return err
	}

	// var tx protoTm.Tx
	// tx.Method = string(fnName)
	// tx.Params = string(paramJSON)
	// tx.Nonce = []byte(nonce)
	// tx.Signature = signature
	// tx.NodeId = ndidID

	// txByte, err := proto.Marshal(&tx)
	// if err != nil {
	// 	return err
	// }

	result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(ndidID))
	// result, err := tmClient.BroadcastTxCommit(txByte)
	if err != nil {
		return err
	}
	log.Printf("SetInitData CheckTx log: %s\n", result.Result.CheckTx.Log)
	log.Printf("SetInitData DeliverTx log: %s\n", result.Result.DeliverTx.Log)

	if result.Result.DeliverTx.Log != "success" {
		// pause 3 sec and retry again
		log.Printf("Retry...\n")
		time.Sleep(3 * time.Second)
		err = setInitData(tmClient, param, ndidKey, ndidID)
		if err != nil {
			return err
		}
	}

	return nil
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
	nonce := base64.StdEncoding.EncodeToString([]byte(tmRand.Str(12)))
	tempPSSmessage := append([]byte(fnName), paramJSON...)
	tempPSSmessage = append(tempPSSmessage, []byte(nonce)...)
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
	tx.Params = string(paramJSON)
	tx.Nonce = []byte(nonce)
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
