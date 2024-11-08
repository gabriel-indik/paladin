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

package zeto

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/hyperledger-labs/zeto/go-sdk/pkg/sparse-merkle-tree/core"
	"github.com/hyperledger-labs/zeto/go-sdk/pkg/sparse-merkle-tree/node"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/kaleido-io/paladin/domains/zeto/internal/msgs"
	"github.com/kaleido-io/paladin/domains/zeto/internal/zeto/smt"
	corepb "github.com/kaleido-io/paladin/domains/zeto/pkg/proto"
	"github.com/kaleido-io/paladin/domains/zeto/pkg/types"
	"github.com/kaleido-io/paladin/domains/zeto/pkg/zetosigner"
	"github.com/kaleido-io/paladin/domains/zeto/pkg/zetosigner/zetosignerapi"
	"github.com/kaleido-io/paladin/toolkit/pkg/algorithms"
	"github.com/kaleido-io/paladin/toolkit/pkg/domain"
	pb "github.com/kaleido-io/paladin/toolkit/pkg/prototk"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
	"github.com/kaleido-io/paladin/toolkit/pkg/verifiers"
	"google.golang.org/protobuf/proto"
)

type transferHandler struct {
	zeto *Zeto
}

var proofComponents = abi.ParameterArray{
	{Name: "pA", Type: "uint256[2]"},
	{Name: "pB", Type: "uint256[2][2]"},
	{Name: "pC", Type: "uint256[2]"},
}

var transferABI = &abi.Entry{
	Type: abi.Function,
	Name: "transfer",
	Inputs: abi.ParameterArray{
		{Name: "inputs", Type: "uint256[]"},
		{Name: "outputs", Type: "uint256[]"},
		{Name: "proof", Type: "tuple", InternalType: "struct Commonlib.Proof", Components: proofComponents},
		{Name: "data", Type: "bytes"},
	},
}

var transferABI_nullifiers = &abi.Entry{
	Type: abi.Function,
	Name: "transfer",
	Inputs: abi.ParameterArray{
		{Name: "nullifiers", Type: "uint256[]"},
		{Name: "outputs", Type: "uint256[]"},
		{Name: "root", Type: "uint256"},
		{Name: "proof", Type: "tuple", InternalType: "struct Commonlib.Proof", Components: proofComponents},
		{Name: "data", Type: "bytes"},
	},
}

var transferABI_withEncryption = &abi.Entry{
	Type: abi.Function,
	Name: "transfer",
	Inputs: abi.ParameterArray{
		{Name: "inputs", Type: "uint256[]"},
		{Name: "outputs", Type: "uint256[]"},
		{Name: "encryptionNonce", Type: "uint256"},
		{Name: "ecdhPublicKey", Type: "uint256[2]"},
		{Name: "encryptedValues", Type: "uint256[]"},
		{Name: "proof", Type: "tuple", InternalType: "struct Commonlib.Proof", Components: proofComponents},
		{Name: "data", Type: "bytes"},
	},
}

var transferABI_withEncryption_nullifiers = &abi.Entry{
	Type: abi.Function,
	Name: "transfer",
	Inputs: abi.ParameterArray{
		{Name: "nullifiers", Type: "uint256[]"},
		{Name: "outputs", Type: "uint256[]"},
		{Name: "root", Type: "uint256"},
		{Name: "encryptionNonce", Type: "uint256"},
		{Name: "ecdhPublicKey", Type: "uint256[2]"},
		{Name: "encryptedValues", Type: "uint256[]"},
		{Name: "proof", Type: "tuple", InternalType: "struct Commonlib.Proof", Components: proofComponents},
		{Name: "data", Type: "bytes"},
	},
}

func (h *transferHandler) ValidateParams(ctx context.Context, config *types.DomainInstanceConfig, params string) (interface{}, error) {
	var transferParams types.TransferParams
	if err := json.Unmarshal([]byte(params), &transferParams); err != nil {
		return nil, err
	}

	if err := validateTransferParams(ctx, transferParams.Transfers); err != nil {
		return nil, err
	}

	return transferParams.Transfers, nil
}

func (h *transferHandler) Init(ctx context.Context, tx *types.ParsedTransaction, req *pb.InitTransactionRequest) (*pb.InitTransactionResponse, error) {
	params := tx.Params.([]*types.TransferParamEntry)

	res := &pb.InitTransactionResponse{
		RequiredVerifiers: []*pb.ResolveVerifierRequest{
			{
				Lookup:       tx.Transaction.From,
				Algorithm:    h.zeto.getAlgoZetoSnarkBJJ(),
				VerifierType: zetosignerapi.IDEN3_PUBKEY_BABYJUBJUB_COMPRESSED_0X,
			},
		},
	}
	for _, param := range params {
		res.RequiredVerifiers = append(res.RequiredVerifiers, &pb.ResolveVerifierRequest{
			Lookup:       param.To,
			Algorithm:    h.zeto.getAlgoZetoSnarkBJJ(),
			VerifierType: zetosignerapi.IDEN3_PUBKEY_BABYJUBJUB_COMPRESSED_0X,
		})
	}

	return res, nil
}

func (h *transferHandler) Assemble(ctx context.Context, tx *types.ParsedTransaction, req *pb.AssembleTransactionRequest) (*pb.AssembleTransactionResponse, error) {
	params := tx.Params.([]*types.TransferParamEntry)

	resolvedSender := domain.FindVerifier(tx.Transaction.From, h.zeto.getAlgoZetoSnarkBJJ(), zetosignerapi.IDEN3_PUBKEY_BABYJUBJUB_COMPRESSED_0X, req.ResolvedVerifiers)
	if resolvedSender == nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorResolveVerifier, tx.Transaction.From)
	}

	useNullifiers := isNullifiersToken(tx.DomainConfig.TokenName)
	inputCoins, inputStates, _, remainder, err := h.zeto.prepareInputs(ctx, useNullifiers, req.StateQueryContext, resolvedSender.Verifier, params)
	if err != nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorPrepTxInputs, err)
	}
	outputCoins, outputStates, err := h.zeto.prepareOutputs(ctx, useNullifiers, params, req.ResolvedVerifiers)
	if err != nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorPrepTxOutputs, err)
	}
	if remainder.Sign() > 0 {
		// add the remainder as an output to the sender themselves
		remainderHex := tktypes.HexUint256(*remainder)
		remainderParams := []*types.TransferParamEntry{
			{
				To:     tx.Transaction.From,
				Amount: &remainderHex,
			},
		}
		returnedCoins, returnedStates, err := h.zeto.prepareOutputs(ctx, useNullifiers, remainderParams, req.ResolvedVerifiers)
		if err != nil {
			return nil, i18n.NewError(ctx, msgs.MsgErrorPrepTxChange, err)
		}
		outputCoins = append(outputCoins, returnedCoins...)
		outputStates = append(outputStates, returnedStates...)
	}

	contractAddress, err := tktypes.ParseEthAddress(req.Transaction.ContractInfo.ContractAddress)
	if err != nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorDecodeContractAddress, err)
	}
	payloadBytes, err := h.formatProvingRequest(ctx, inputCoins, outputCoins, tx.DomainConfig.CircuitId, tx.DomainConfig.TokenName, req.StateQueryContext, contractAddress)
	if err != nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorFormatProvingReq, err)
	}

	return &pb.AssembleTransactionResponse{
		AssemblyResult: pb.AssembleTransactionResponse_OK,
		AssembledTransaction: &pb.AssembledTransaction{
			InputStates:  inputStates,
			OutputStates: outputStates,
		},
		AttestationPlan: []*pb.AttestationRequest{
			{
				Name:            "sender",
				AttestationType: pb.AttestationType_SIGN,
				Algorithm:       h.zeto.getAlgoZetoSnarkBJJ(),
				VerifierType:    zetosignerapi.IDEN3_PUBKEY_BABYJUBJUB_COMPRESSED_0X,
				PayloadType:     zetosignerapi.PAYLOAD_DOMAIN_ZETO_SNARK,
				Payload:         payloadBytes,
				Parties:         []string{tx.Transaction.From},
			},
			{
				Name:            "submitter",
				AttestationType: pb.AttestationType_ENDORSE,
				Algorithm:       algorithms.ECDSA_SECP256K1,
				VerifierType:    verifiers.ETH_ADDRESS,
				Parties:         []string{tx.Transaction.From},
			},
		},
	}, nil
}

func (h *transferHandler) Endorse(ctx context.Context, tx *types.ParsedTransaction, req *pb.EndorseTransactionRequest) (*pb.EndorseTransactionResponse, error) {
	return &pb.EndorseTransactionResponse{
		EndorsementResult: pb.EndorseTransactionResponse_ENDORSER_SUBMIT,
	}, nil
}

func getTransferABI(tokenName string) *abi.Entry {
	transferFunction := transferABI
	if isEncryptionToken(tokenName) {
		transferFunction = transferABI_withEncryption
		if isNullifiersToken(tokenName) {
			transferFunction = transferABI_withEncryption_nullifiers
		}
	} else if isNullifiersToken(tokenName) {
		transferFunction = transferABI_nullifiers
	}
	return transferFunction
}

func (h *transferHandler) Prepare(ctx context.Context, tx *types.ParsedTransaction, req *pb.PrepareTransactionRequest) (*pb.PrepareTransactionResponse, error) {
	var proofRes corepb.ProvingResponse
	result := domain.FindAttestation("sender", req.AttestationResult)
	if result == nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorFindSenderAttestation)
	}
	if err := proto.Unmarshal(result.Payload, &proofRes); err != nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorUnmarshalProvingRes, err)
	}

	inputSize := getInputSize(len(req.InputStates))
	inputs := make([]string, inputSize)
	for i := 0; i < inputSize; i++ {
		if i < len(req.InputStates) {
			state := req.InputStates[i]
			coin, err := h.zeto.makeCoin(state.StateDataJson)
			if err != nil {
				return nil, i18n.NewError(ctx, msgs.MsgErrorParseInputStates, err)
			}
			hash, err := coin.Hash(ctx)
			if err != nil {
				return nil, i18n.NewError(ctx, msgs.MsgErrorHashInputState, err)
			}
			inputs[i] = hash.String()
		} else {
			inputs[i] = "0"
		}
	}
	outputs := make([]string, inputSize)
	for i := 0; i < inputSize; i++ {
		if i < len(req.OutputStates) {
			state := req.OutputStates[i]
			coin, err := h.zeto.makeCoin(state.StateDataJson)
			if err != nil {
				return nil, i18n.NewError(ctx, msgs.MsgErrorParseOutputStates, err)
			}
			hash, err := coin.Hash(ctx)
			if err != nil {
				return nil, i18n.NewError(ctx, msgs.MsgErrorHashOutputState, err)
			}
			outputs[i] = hash.String()
		} else {
			outputs[i] = "0"
		}
	}

	data, err := encodeTransactionData(ctx, req.Transaction)
	if err != nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorEncodeTxData, err)
	}
	params := map[string]any{
		"inputs":  inputs,
		"outputs": outputs,
		"proof":   h.encodeProof(proofRes.Proof),
		"data":    data,
	}
	transferFunction := getTransferABI(tx.DomainConfig.TokenName)
	if isEncryptionToken(tx.DomainConfig.TokenName) {
		params["ecdhPublicKey"] = strings.Split(proofRes.PublicInputs["ecdhPublicKey"], ",")
		params["encryptionNonce"] = proofRes.PublicInputs["encryptionNonce"]
		params["encryptedValues"] = strings.Split(proofRes.PublicInputs["encryptedValues"], ",")
	}
	if isNullifiersToken(tx.DomainConfig.TokenName) {
		delete(params, "inputs")
		params["nullifiers"] = strings.Split(proofRes.PublicInputs["nullifiers"], ",")
		params["root"] = proofRes.PublicInputs["root"]
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorMarshalPrepedParams, err)
	}
	functionJSON, err := json.Marshal(transferFunction)
	if err != nil {
		return nil, err
	}

	return &pb.PrepareTransactionResponse{
		Transaction: &pb.PreparedTransaction{
			FunctionAbiJson: string(functionJSON),
			ParamsJson:      string(paramsJSON),
		},
	}, nil
}

func (h *transferHandler) formatProvingRequest(ctx context.Context, inputCoins, outputCoins []*types.ZetoCoin, circuitId, tokenName, stateQueryContext string, contractAddress *tktypes.EthAddress) ([]byte, error) {
	inputSize := getInputSize(len(inputCoins))
	inputCommitments := make([]string, inputSize)
	inputValueInts := make([]uint64, inputSize)
	inputSalts := make([]string, inputSize)
	inputOwner := inputCoins[0].Owner.String()
	for i := 0; i < inputSize; i++ {
		if i < len(inputCoins) {
			coin := inputCoins[i]
			hash, err := coin.Hash(ctx)
			if err != nil {
				return nil, i18n.NewError(ctx, msgs.MsgErrorHashInputState, err)
			}
			inputCommitments[i] = hash.Int().Text(16)
			inputValueInts[i] = coin.Amount.Int().Uint64()
			inputSalts[i] = coin.Salt.Int().Text(16)
		} else {
			inputCommitments[i] = "0"
			inputSalts[i] = "0"
		}
	}

	outputValueInts := make([]uint64, inputSize)
	outputSalts := make([]string, inputSize)
	outputOwners := make([]string, inputSize)
	for i := 0; i < inputSize; i++ {
		if i < len(outputCoins) {
			coin := outputCoins[i]
			outputValueInts[i] = coin.Amount.Int().Uint64()
			outputSalts[i] = coin.Salt.Int().Text(16)
			outputOwners[i] = coin.Owner.String()
		} else {
			outputSalts[i] = "0"
		}
	}

	var extras []byte
	if isNullifiersCircuit(circuitId) {
		proofs, extrasObj, err := h.generateMerkleProofs(ctx, tokenName, stateQueryContext, contractAddress, inputCoins)
		if err != nil {
			return nil, i18n.NewError(ctx, msgs.MsgErrorGenerateMTP, err)
		}
		for i := len(proofs); i < inputSize; i++ {
			extrasObj.MerkleProofs = append(extrasObj.MerkleProofs, &smt.Empty_Proof)
			extrasObj.Enabled = append(extrasObj.Enabled, false)
		}
		protoExtras, err := proto.Marshal(extrasObj)
		if err != nil {
			return nil, i18n.NewError(ctx, msgs.MsgErrorMarshalExtraObj, err)
		}
		extras = protoExtras
	}

	payload := &corepb.ProvingRequest{
		CircuitId: circuitId,
		Common: &corepb.ProvingRequestCommon{
			InputCommitments: inputCommitments,
			InputValues:      inputValueInts,
			InputSalts:       inputSalts,
			InputOwner:       inputOwner,
			OutputValues:     outputValueInts,
			OutputSalts:      outputSalts,
			OutputOwners:     outputOwners,
		},
	}
	if extras != nil {
		payload.Extras = extras
	}
	return proto.Marshal(payload)
}

func (h *transferHandler) encodeProof(proof *corepb.SnarkProof) map[string]interface{} {
	// Convert the proof json to the format that the Solidity verifier expects
	return map[string]interface{}{
		"pA": []string{proof.A[0], proof.A[1]},
		"pB": [][]string{
			{proof.B[0].Items[1], proof.B[0].Items[0]},
			{proof.B[1].Items[1], proof.B[1].Items[0]},
		},
		"pC": []string{proof.C[0], proof.C[1]},
	}
}

func (h *transferHandler) generateMerkleProofs(ctx context.Context, tokenName string, stateQueryContext string, contractAddress *tktypes.EthAddress, inputCoins []*types.ZetoCoin) ([]core.Proof, *corepb.ProvingRequestExtras_Nullifiers, error) {
	smtName := smt.MerkleTreeName(tokenName, contractAddress)
	storage := smt.NewStatesStorage(h.zeto.Callbacks, smtName, stateQueryContext, h.zeto.merkleTreeRootSchema.Id, h.zeto.merkleTreeNodeSchema.Id)
	mt, err := smt.NewSmt(storage)
	if err != nil {
		return nil, nil, i18n.NewError(ctx, msgs.MsgErrorNewSmt, smtName, err)
	}
	// verify that the input UTXOs have been indexed by the Merkle tree DB
	// and generate a merkle proof for each
	var indexes []*big.Int
	for _, coin := range inputCoins {
		pubKey, err := zetosigner.DecodeBabyJubJubPublicKey(coin.Owner.String())
		if err != nil {
			return nil, nil, i18n.NewError(ctx, msgs.MsgErrorLoadOwnerPubKey, err)
		}
		idx := node.NewFungible(coin.Amount.Int(), pubKey, coin.Salt.Int())
		leaf, err := node.NewLeafNode(idx)
		if err != nil {
			return nil, nil, i18n.NewError(ctx, msgs.MsgErrorNewLeafNode, err)
		}
		n, err := mt.GetNode(leaf.Ref())
		if err != nil {
			// TODO: deal with when the node is not found in the DB tables for the tree
			// e.g because the transaction event hasn't been processed yet
			return nil, nil, i18n.NewError(ctx, msgs.MsgErrorQueryLeafNode, leaf.Ref().Hex(), err)
		}
		hash, err := coin.Hash(ctx)
		if err != nil {
			return nil, nil, i18n.NewError(ctx, msgs.MsgErrorHashInputState, err)
		}
		if n.Index().BigInt().Cmp(hash.Int()) != 0 {
			expectedIndex, err := node.NewNodeIndexFromBigInt(hash.Int())
			if err != nil {
				return nil, nil, i18n.NewError(ctx, msgs.MsgErrorNewNodeIndex, err)
			}
			return nil, nil, i18n.NewError(ctx, msgs.MsgErrorHashMismatch, leaf.Ref().Hex(), n.Index().BigInt().Text(16), n.Index().Hex(), hash.HexString0xPrefix(), expectedIndex.Hex())
		}
		indexes = append(indexes, n.Index().BigInt())
	}
	mtRoot := mt.Root()
	proofs, _, err := mt.GenerateProofs(indexes, mtRoot)
	if err != nil {
		return nil, nil, i18n.NewError(ctx, msgs.MsgErrorGenerateMTP, err)
	}
	var mps []*corepb.MerkleProof
	var enabled []bool
	for i, proof := range proofs {
		cp, err := proof.ToCircomVerifierProof(indexes[i], indexes[i], mtRoot, smt.SMT_HEIGHT_UTXO)
		if err != nil {
			return nil, nil, i18n.NewError(ctx, msgs.MsgErrorConvertToCircomProof, err)
		}
		proofSiblings := make([]string, len(cp.Siblings)-1)
		for i, s := range cp.Siblings[0 : len(cp.Siblings)-1] {
			proofSiblings[i] = s.BigInt().Text(16)
		}
		p := corepb.MerkleProof{
			Nodes: proofSiblings,
		}
		mps = append(mps, &p)
		enabled = append(enabled, true)
	}
	extrasObj := corepb.ProvingRequestExtras_Nullifiers{
		Root:         mt.Root().BigInt().Text(16),
		MerkleProofs: mps,
		Enabled:      enabled,
	}

	return proofs, &extrasObj, nil
}
