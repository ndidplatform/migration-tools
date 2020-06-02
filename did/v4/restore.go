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

package v4

import (
	"bufio"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	tmRand "github.com/ndidplatform/migration-tools/rand"

	"github.com/ndidplatform/migration-tools/utils"
)

func Restore(
	ndidID string,
	backupDataDir string,
	backupDataFileName string,
	chainHistoryFileName string,
	keyDir string,
	tendermintRPCAddress string,
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
	dataMaster, err := ioutil.ReadAll(ndidMasterKeyFile)
	if err != nil {
		return err
	}
	ndidPrivKey := utils.GetPrivateKeyFromString(string(data))
	ndidMasterPrivKey := utils.GetPrivateKeyFromString(string(dataMaster))
	err = initNDID(
		ndidPrivKey,
		ndidMasterPrivKey,
		ndidID,
		backupDataDir,
		chainHistoryFileName,
		tendermintRPCAddress,
	)
	if err != nil {
		return err
	}
	file, err := os.Open(backupDataDir + backupDataFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	maximumBytes := 100000
	size := 0
	count := 0
	nTx := 0

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
		if size > maximumBytes {
			err = setInitData(param, ndidPrivKey, ndidID, tendermintRPCAddress)
			if err != nil {
				return err
			}
			fmt.Print("Number of kv in param: ")
			fmt.Println(count)
			fmt.Print("Total number of kv: ")
			fmt.Println(nTx)
			count = 0
			size = 0
			param.KVList = make([]KeyValue, 0)
		}
	}
	if count > 0 {
		err = setInitData(param, ndidPrivKey, ndidID, tendermintRPCAddress)
		if err != nil {
			return err
		}
		fmt.Print("Number of kv in param: ")
		fmt.Println(count)
		fmt.Print("Total number of kv: ")
		fmt.Println(nTx)
	}
	err = endInit(ndidPrivKey, ndidID, tendermintRPCAddress)
	if err != nil {
		return err
	}

	return nil
}

func initNDID(
	ndidKey *rsa.PrivateKey,
	ndidMasterKey *rsa.PrivateKey,
	ndidID string,
	backupDataDir string,
	chainHistoryFileName string,
	tendermintRPCAddress string,
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
	result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(initNDIDparam.NodeID))
	if err != nil {
		return err
	}
	fmt.Println(result.Result.DeliverTx.Log)

	return nil
}

func setInitData(param SetInitDataParam, ndidKey *rsa.PrivateKey, ndidID string, tendermintRPCAddress string) (err error) {
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
	result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(ndidID))
	if err != nil {
		return err
	}
	fmt.Printf("CheckTx log: %s\n", result.Result.CheckTx.Log)
	fmt.Printf("DeliverTx log: %s\n", result.Result.DeliverTx.Log)

	if result.Result.DeliverTx.Log != "success" {
		// pause 3 sec and retry again
		fmt.Printf("Retry ...\n")
		time.Sleep(3 * time.Second)
		err = setInitData(param, ndidKey, ndidID, tendermintRPCAddress)
		if err != nil {
			return err
		}
	}

	return nil
}

func endInit(ndidKey *rsa.PrivateKey, ndidID string, tendermintRPCAddress string) (err error) {
	var param EndInitParam
	paramJSON, err := json.Marshal(param)
	if err != nil {
		return err
	}
	fnName := "EndInit"
	nodeID := ndidID
	nonce := base64.StdEncoding.EncodeToString([]byte(tmRand.Str(12)))
	tempPSSmessage := append([]byte(fnName), paramJSON...)
	tempPSSmessage = append(tempPSSmessage, []byte(nonce)...)
	PSSmessage := []byte(base64.StdEncoding.EncodeToString(tempPSSmessage))
	newhash := crypto.SHA256
	pssh := newhash.New()
	pssh.Write(PSSmessage)
	hashed := pssh.Sum(nil)
	signature, err := rsa.SignPKCS1v15(rand.Reader, ndidKey, newhash, hashed)
	result, err := CallTendermint(tendermintRPCAddress, []byte(fnName), paramJSON, []byte(nonce), signature, []byte(nodeID))
	if err != nil {
		return err
	}
	fmt.Println(result.Result.DeliverTx.Log)

	return nil
}
