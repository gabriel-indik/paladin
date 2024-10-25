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

package types

import (
	"context"

	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/kaleido-io/paladin/toolkit/pkg/pldapi"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
)

var NotoABI = abi.ABI{
	{
		Name: "mint",
		Type: abi.Function,
		Inputs: abi.ParameterArray{
			{Name: "to", Type: "string"},
			{Name: "amount", Type: "uint256"},
		},
	},
	{
		Name: "transfer",
		Type: abi.Function,
		Inputs: abi.ParameterArray{
			{Name: "to", Type: "string"},
			{Name: "amount", Type: "uint256"},
		},
	},
	{
		Name: "approveTransfer",
		Type: abi.Function,
		Inputs: abi.ParameterArray{
			{
				Name:         "inputs",
				Type:         "tuple[]",
				InternalType: "struct FullState[]",
				Components: abi.ParameterArray{
					{Name: "id", Type: "bytes"},
					{Name: "schema", Type: "bytes32"},
					{Name: "data", Type: "bytes"},
				},
			},
			{
				Name:         "outputs",
				Type:         "tuple[]",
				InternalType: "struct FullState[]",
				Components: abi.ParameterArray{
					{Name: "id", Type: "bytes"},
					{Name: "schema", Type: "bytes32"},
					{Name: "data", Type: "bytes"},
				},
			},
			{Name: "data", Type: "bytes"},
			{Name: "delegate", Type: "address"},
		},
	},
}

type ConstructorParams struct {
	Notary          string      `json:"notary"`                    // Lookup string for the notary identity
	Implementation  string      `json:"implementation,omitempty"`  // Use a specific implementation of Noto that was registered to the factory (blank to use default)
	Hooks           *HookParams `json:"hooks,omitempty"`           // Configure hooks for programmable logic around Noto operations
	RestrictMinting *bool       `json:"restrictMinting,omitempty"` // Only allow notary to mint (default: true)
}

// Currently the only supported hooks are provided via a Pente private smart contract
type HookParams struct {
	PrivateGroup   *PentePrivateGroup  `json:"privateGroup,omitempty"`   // Details on a Pente privacy group
	PublicAddress  *tktypes.EthAddress `json:"publicAddress,omitempty"`  // Public address of the Pente privacy group
	PrivateAddress *tktypes.EthAddress `json:"privateAddress,omitempty"` // Private address of the hook contract deployed within the privacy group
}

type MintParams struct {
	To     string              `json:"to"`
	Amount *tktypes.HexUint256 `json:"amount"`
}

type TransferParams struct {
	To     string              `json:"to"`
	Amount *tktypes.HexUint256 `json:"amount"`
}

type ApproveParams struct {
	Inputs   []*pldapi.StateWithData `json:"inputs"`
	Outputs  []*pldapi.StateWithData `json:"outputs"`
	Data     tktypes.HexBytes        `json:"data"`
	Delegate *tktypes.EthAddress     `json:"delegate"`
}

type ApproveExtraParams struct {
	Data tktypes.HexBytes `json:"data"`
}

type NotoPublicTransaction struct {
	FunctionABI *abi.Entry      `json:"functionABI"`
	ParamsJSON  tktypes.RawJSON `json:"paramsJSON"`
	EncodedCall []byte          `json:"encodedCall"`
}

func (npt *NotoPublicTransaction) Encode(ctx context.Context) ([]byte, error) {
	return npt.FunctionABI.EncodeCallDataJSONCtx(ctx, npt.ParamsJSON)
}

type NotoTransferMetadata struct {
	ApprovalParams       ApproveExtraParams    `json:"approvalParams"`       // Partial set of params that can be passed to the "approveTransfer" method to approve another party to perform this transfer
	TransferWithApproval NotoPublicTransaction `json:"transferWithApproval"` // The public transaction that would need to be submitted by an approved party to perform this transfer
}
