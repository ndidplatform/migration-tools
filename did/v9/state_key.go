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

package v9

var (
	MasterNDIDKeyBytes                            = []byte("MasterNDID")
	InitStateKeyBytes                             = []byte("InitState")
	LastBlockKeyBytes                             = []byte("lastBlock")
	IdpListKeyBytes                               = []byte("IdPList")
	AllNamespaceKeyBytes                          = []byte("AllNamespace")
	ServicePriceMinEffectiveDatetimeDelayKeyBytes = []byte("ServicePriceMinEffectiveDatetimeDelay")
	SupportedIALListKeyBytes                      = []byte("SupportedIALList")
	SupportedAALListKeyBytes                      = []byte("SupportedAALList")
)

const (
	KeySeparator                                         = "|"
	NonceKeyPrefix                                       = "n"
	NodeIDKeyPrefix                                      = "NodeID"
	NodeKeyKeyPrefix                                     = "NodeKey"
	BehindProxyNodeKeyPrefix                             = "BehindProxyNode"
	TokenKeyPrefix                                       = "Token"
	TokenPriceFuncKeyPrefix                              = "TokenPriceFunc"
	ServiceKeyPrefix                                     = "Service"
	ServiceDestinationKeyPrefix                          = "ServiceDestination"
	ApprovedServiceKeyPrefix                             = "ApproveKey"
	ProvidedServicesKeyPrefix                            = "ProvideService"
	RefGroupCodeKeyPrefix                                = "RefGroupCode"
	IdentityToRefCodeKeyPrefix                           = "identityToRefCodeKey"
	AccessorToRefCodeKeyPrefix                           = "accessorToRefCodeKey"
	AllowedModeListKeyPrefix                             = "AllowedModeList"
	RequestKeyPrefix                                     = "Request"
	MessageKeyPrefix                                     = "Message"
	DataSignatureKeyPrefix                               = "SignData"
	ErrorCodeKeyPrefix                                   = "ErrorCode"
	ErrorCodeListKeyPrefix                               = "ErrorCodeList"
	ServicePriceCeilingKeyPrefix                         = "ServicePriceCeiling"
	ServicePriceMinEffectiveDatetimeDelayKeyPrefix       = "ServicePriceMinEffectiveDatetimeDelay"
	ServicePriceListKeyPrefix                            = "ServicePriceListKey"
	RequestTypeKeyPrefix                                 = "RequestType"
	SuppressedIdentityModificationNotificationNodePrefix = "SuppressedIdentityModificationNotificationNode"
	NodeSupportedFeatureKeyPrefix                        = "NodeSupportedFeature"
	ValidatorKeyPrefix                                   = "Validator"
)
