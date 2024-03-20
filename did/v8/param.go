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

package v8

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

type UpdateNodeParam struct {
	PublicKey                              string   `json:"public_key"`
	MasterPublicKey                        string   `json:"master_public_key"`
	SupportedRequestMessageDataUrlTypeList []string `json:"supported_request_message_data_url_type_list"`
}
