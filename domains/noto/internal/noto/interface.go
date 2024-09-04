/*
 * Copyright © 2024 Kaleido, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
 * an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package noto

import (
	"context"

	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	pb "github.com/kaleido-io/paladin/toolkit/pkg/prototk"
)

type DomainInterface map[string]*DomainEntry

type DomainEntry struct {
	ABI     *abi.Entry
	handler DomainHandler
}

type DomainHandler interface {
	ValidateParams(params string) (interface{}, error)
	Init(ctx context.Context, tx *parsedTransaction, req *pb.InitTransactionRequest) (*pb.InitTransactionResponse, error)
	Assemble(ctx context.Context, tx *parsedTransaction, req *pb.AssembleTransactionRequest) (*pb.AssembleTransactionResponse, error)
	Endorse(ctx context.Context, tx *parsedTransaction, req *pb.EndorseTransactionRequest) (*pb.EndorseTransactionResponse, error)
	Prepare(ctx context.Context, tx *parsedTransaction, req *pb.PrepareTransactionRequest) (*pb.PrepareTransactionResponse, error)
}

type domainHandler struct {
	noto *Noto
}

func (n *Noto) getInterface() DomainInterface {
	iface := DomainInterface{
		"constructor": {
			ABI: &abi.Entry{
				Type: abi.Constructor,
				Inputs: abi.ParameterArray{
					{Name: "notary", Type: "string"},
				},
			},
		},
		"mint": {
			ABI: &abi.Entry{
				Name: "mint",
				Type: abi.Function,
				Inputs: abi.ParameterArray{
					{Name: "to", Type: "string"},
					{Name: "amount", Type: "uint256"},
				},
			},
		},
		"transfer": {
			ABI: &abi.Entry{
				Name: "transfer",
				Type: abi.Function,
				Inputs: abi.ParameterArray{
					{Name: "to", Type: "string"},
					{Name: "amount", Type: "uint256"},
				},
			},
		},
	}

	iface["mint"].handler = &mintHandler{
		domainHandler: domainHandler{noto: n},
	}
	iface["transfer"].handler = &transferHandler{
		domainHandler: domainHandler{noto: n},
	}

	return iface
}

type NotoConstructorParams struct {
	Notary string `json:"notary"`
}

type NotoMintParams struct {
	To     string               `json:"to"`
	Amount *ethtypes.HexInteger `json:"amount"`
}

type NotoTransferParams struct {
	To     string               `json:"to"`
	Amount *ethtypes.HexInteger `json:"amount"`
}