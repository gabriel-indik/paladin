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
	_ "embed"
	"encoding/json"
	"math/big"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"github.com/kaleido-io/paladin/domains/noto/internal/msgs"
	"github.com/kaleido-io/paladin/domains/noto/pkg/types"
	"github.com/kaleido-io/paladin/toolkit/pkg/algorithms"
	"github.com/kaleido-io/paladin/toolkit/pkg/domain"
	"github.com/kaleido-io/paladin/toolkit/pkg/plugintk"
	"github.com/kaleido-io/paladin/toolkit/pkg/prototk"
	"github.com/kaleido-io/paladin/toolkit/pkg/solutils"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
	"github.com/kaleido-io/paladin/toolkit/pkg/verifiers"
)

//go:embed abis/NotoFactory.json
var notoFactoryJSON []byte

//go:embed abis/INoto.json
var notoInterfaceJSON []byte

//go:embed abis/INotoHooks.json
var notoHooksJSON []byte

func NewNoto(callbacks plugintk.DomainCallbacks) plugintk.DomainAPI {
	return &Noto{Callbacks: callbacks}
}

type Noto struct {
	Callbacks plugintk.DomainCallbacks

	name              string
	config            types.DomainConfig
	chainID           int64
	coinSchema        *prototk.StateSchema
	factoryABI        abi.ABI
	contractABI       abi.ABI
	transferSignature string
	approvedSignature string
}

type NotoDeployParams struct {
	Name          string             `json:"name,omitempty"`
	TransactionID string             `json:"transactionId"`
	NotaryAddress tktypes.EthAddress `json:"notaryAddress"`
	Data          tktypes.HexBytes   `json:"data"`
}

type NotoMintParams struct {
	Outputs   []string         `json:"outputs"`
	Signature tktypes.HexBytes `json:"signature"`
	Data      tktypes.HexBytes `json:"data"`
}

type NotoTransferParams struct {
	Inputs    []string         `json:"inputs"`
	Outputs   []string         `json:"outputs"`
	Signature tktypes.HexBytes `json:"signature"`
	Data      tktypes.HexBytes `json:"data"`
}

type NotoApproveTransferParams struct {
	Delegate  *tktypes.EthAddress `json:"delegate"`
	TXHash    tktypes.HexBytes    `json:"txhash"`
	Signature tktypes.HexBytes    `json:"signature"`
	Data      tktypes.HexBytes    `json:"data"`
}

type NotoTransfer_Event struct {
	Inputs    []tktypes.Bytes32 `json:"inputs"`
	Outputs   []tktypes.Bytes32 `json:"outputs"`
	Signature tktypes.HexBytes  `json:"signature"`
	Data      tktypes.HexBytes  `json:"data"`
}

type NotoApproved_Event struct {
	Delegate  tktypes.EthAddress `json:"delegate"`
	TXHash    tktypes.Bytes32    `json:"txhash"`
	Signature tktypes.HexBytes   `json:"signature"`
	Data      tktypes.HexBytes   `json:"data"`
}

type gatheredCoins struct {
	inCoins   []*types.NotoCoin
	inStates  []*prototk.StateRef
	inTotal   *big.Int
	outCoins  []*types.NotoCoin
	outStates []*prototk.StateRef
	outTotal  *big.Int
}

func getEventSignature(ctx context.Context, abi abi.ABI, eventName string) (string, error) {
	event := abi.Events()[eventName]
	if event == nil {
		return "", i18n.NewError(ctx, msgs.MsgUnknownEvent, eventName)
	}
	return event.SolString(), nil
}

func (n *Noto) Name() string {
	return n.name
}

func (n *Noto) CoinSchemaID() string {
	return n.coinSchema.Id
}

func (n *Noto) ConfigureDomain(ctx context.Context, req *prototk.ConfigureDomainRequest) (*prototk.ConfigureDomainResponse, error) {
	err := json.Unmarshal([]byte(req.ConfigJson), &n.config)
	if err != nil {
		return nil, err
	}

	factory := solutils.MustLoadBuild(notoFactoryJSON)
	contract := solutils.MustLoadBuild(notoInterfaceJSON)

	n.name = req.Name
	n.chainID = req.ChainId
	n.factoryABI = factory.ABI
	n.contractABI = contract.ABI

	n.transferSignature, err = getEventSignature(ctx, contract.ABI, "NotoTransfer")
	if err != nil {
		return nil, err
	}
	n.approvedSignature, err = getEventSignature(ctx, contract.ABI, "NotoApproved")
	if err != nil {
		return nil, err
	}

	schemaJSON, err := json.Marshal(types.NotoCoinABI)
	if err != nil {
		return nil, err
	}

	var events abi.ABI
	for _, entry := range contract.ABI {
		if entry.Type == abi.Event {
			events = append(events, entry)
		}
	}
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return nil, err
	}

	return &prototk.ConfigureDomainResponse{
		DomainConfig: &prototk.DomainConfig{
			AbiStateSchemasJson: []string{string(schemaJSON)},
			AbiEventsJson:       string(eventsJSON),
		},
	}, nil
}

func (n *Noto) InitDomain(ctx context.Context, req *prototk.InitDomainRequest) (*prototk.InitDomainResponse, error) {
	n.coinSchema = req.AbiStateSchemas[0]
	return &prototk.InitDomainResponse{}, nil
}

func (n *Noto) InitDeploy(ctx context.Context, req *prototk.InitDeployRequest) (*prototk.InitDeployResponse, error) {
	params, err := n.validateDeploy(req.Transaction)
	if err != nil {
		return nil, err
	}
	return &prototk.InitDeployResponse{
		RequiredVerifiers: []*prototk.ResolveVerifierRequest{
			{
				Lookup:       params.Notary,
				Algorithm:    algorithms.ECDSA_SECP256K1,
				VerifierType: verifiers.ETH_ADDRESS,
			},
		},
	}, nil
}

func (n *Noto) PrepareDeploy(ctx context.Context, req *prototk.PrepareDeployRequest) (*prototk.PrepareDeployResponse, error) {
	params, err := n.validateDeploy(req.Transaction)
	if err != nil {
		return nil, err
	}
	notary := domain.FindVerifier(params.Notary, algorithms.ECDSA_SECP256K1, verifiers.ETH_ADDRESS, req.ResolvedVerifiers)
	if notary == nil {
		return nil, i18n.NewError(ctx, msgs.MsgErrorVerifyingAddress, "notary")
	}

	deployData := &types.NotoConfigData_V0{
		NotaryLookup:    notary.Lookup,
		NotaryType:      types.NotaryTypeSigner,
		RestrictMinting: true,
	}
	if params.RestrictMinting != nil {
		deployData.RestrictMinting = *params.RestrictMinting
	}

	notaryAddress, err := tktypes.ParseEthAddress(notary.Verifier)
	if err != nil {
		return nil, err
	}

	if params.Hooks != nil && !params.Hooks.PublicAddress.IsZero() {
		notaryAddress = params.Hooks.PublicAddress
		deployData.NotaryType = types.NotaryTypePente
		deployData.PrivateAddress = params.Hooks.PrivateAddress
		deployData.PrivateGroup = params.Hooks.PrivateGroup
	}

	deployDataJSON, err := json.Marshal(deployData)
	if err != nil {
		return nil, err
	}
	deployParams := &NotoDeployParams{
		Name:          params.Implementation,
		TransactionID: req.Transaction.TransactionId,
		NotaryAddress: *notaryAddress,
		Data:          deployDataJSON,
	}
	paramsJSON, err := json.Marshal(deployParams)
	if err != nil {
		return nil, err
	}
	functionName := "deploy"
	if deployParams.Name != "" {
		functionName = "deployImplementation"
	}
	functionJSON, err := json.Marshal(n.factoryABI.Functions()[functionName])
	if err != nil {
		return nil, err
	}

	return &prototk.PrepareDeployResponse{
		Transaction: &prototk.PreparedTransaction{
			FunctionAbiJson: string(functionJSON),
			ParamsJson:      string(paramsJSON),
		},
		Signer: &notary.Lookup,
	}, nil
}

func (n *Noto) InitContract(ctx context.Context, req *prototk.InitContractRequest) (*prototk.InitContractResponse, error) {
	var notoContractConfigJSON []byte
	var staticCoordinator string
	domainConfig, err := n.decodeConfig(ctx, req.ContractConfig)
	if err == nil {
		parsedConfig := &types.NotoParsedConfig{
			NotaryType:      domainConfig.DecodedData.NotaryType,
			NotaryAddress:   domainConfig.NotaryAddress,
			Variant:         domainConfig.Variant,
			NotaryLookup:    domainConfig.DecodedData.NotaryLookup,
			PrivateAddress:  domainConfig.DecodedData.PrivateAddress,
			PrivateGroup:    domainConfig.DecodedData.PrivateGroup,
			RestrictMinting: domainConfig.DecodedData.RestrictMinting,
		}
		notoContractConfigJSON, err = json.Marshal(parsedConfig)
	}
	if err == nil {
		staticCoordinator = domainConfig.DecodedData.NotaryLookup
	}
	if err != nil {
		// This on-chain contract has invalid configuration - not an error in our process
		return &prototk.InitContractResponse{Valid: false}, nil
	}

	return &prototk.InitContractResponse{
		Valid: true,
		ContractConfig: &prototk.ContractConfig{
			ContractConfigJson:   string(notoContractConfigJSON),
			CoordinatorSelection: prototk.ContractConfig_COORDINATOR_STATIC,
			StaticCoordinator:    &staticCoordinator,
			SubmitterSelection:   prototk.ContractConfig_SUBMITTER_COORDINATOR,
		},
	}, nil
}

func (n *Noto) InitTransaction(ctx context.Context, req *prototk.InitTransactionRequest) (*prototk.InitTransactionResponse, error) {
	tx, handler, err := n.validateTransaction(ctx, req.Transaction)
	if err != nil {
		return nil, err
	}
	return handler.Init(ctx, tx, req)
}

func (n *Noto) AssembleTransaction(ctx context.Context, req *prototk.AssembleTransactionRequest) (*prototk.AssembleTransactionResponse, error) {
	tx, handler, err := n.validateTransaction(ctx, req.Transaction)
	if err != nil {
		return nil, err
	}
	return handler.Assemble(ctx, tx, req)
}

func (n *Noto) EndorseTransaction(ctx context.Context, req *prototk.EndorseTransactionRequest) (*prototk.EndorseTransactionResponse, error) {
	tx, handler, err := n.validateTransaction(ctx, req.Transaction)
	if err != nil {
		return nil, err
	}
	return handler.Endorse(ctx, tx, req)
}

func (n *Noto) PrepareTransaction(ctx context.Context, req *prototk.PrepareTransactionRequest) (*prototk.PrepareTransactionResponse, error) {
	tx, handler, err := n.validateTransaction(ctx, req.Transaction)
	if err != nil {
		return nil, err
	}
	return handler.Prepare(ctx, tx, req)
}

func (n *Noto) decodeConfig(ctx context.Context, domainConfig []byte) (*types.NotoConfig_V0, error) {
	configSelector := ethtypes.HexBytes0xPrefix(domainConfig[0:4])
	if configSelector.String() != types.NotoConfigID_V0.String() {
		return nil, i18n.NewError(ctx, msgs.MsgUnexpectedConfigType, configSelector)
	}
	configValues, err := types.NotoConfigABI_V0.DecodeABIDataCtx(ctx, domainConfig[4:], 0)
	if err != nil {
		return nil, err
	}
	configJSON, err := tktypes.StandardABISerializer().SerializeJSON(configValues)
	if err != nil {
		return nil, err
	}
	var config types.NotoConfig_V0
	err = json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(config.Data, &config.DecodedData)
	return &config, err
}

func (n *Noto) validateDeploy(tx *prototk.DeployTransactionSpecification) (*types.ConstructorParams, error) {
	var params types.ConstructorParams
	err := json.Unmarshal([]byte(tx.ConstructorParamsJson), &params)
	return &params, err
}

func (n *Noto) validateTransaction(ctx context.Context, tx *prototk.TransactionSpecification) (*types.ParsedTransaction, types.DomainHandler, error) {
	var functionABI abi.Entry
	err := json.Unmarshal([]byte(tx.FunctionAbiJson), &functionABI)
	if err != nil {
		return nil, nil, err
	}

	var domainConfig *types.NotoParsedConfig
	err = json.Unmarshal([]byte(tx.ContractInfo.ContractConfigJson), &domainConfig)
	if err != nil {
		return nil, nil, err
	}

	abi := types.NotoABI.Functions()[functionABI.Name]
	handler := n.GetHandler(functionABI.Name)
	if abi == nil || handler == nil {
		return nil, nil, i18n.NewError(ctx, msgs.MsgUnknownFunction, functionABI.Name)
	}
	params, err := handler.ValidateParams(ctx, domainConfig, tx.FunctionParamsJson)
	if err != nil {
		return nil, nil, err
	}

	signature, err := abi.SolidityStringCtx(ctx)
	if err != nil {
		return nil, nil, err
	}
	if tx.FunctionSignature != signature {
		return nil, nil, i18n.NewError(ctx, msgs.MsgUnexpectedFunctionSignature, functionABI.Name, signature, tx.FunctionSignature)
	}

	contractAddress, err := ethtypes.NewAddress(tx.ContractInfo.ContractAddress)
	if err != nil {
		return nil, nil, err
	}

	return &types.ParsedTransaction{
		Transaction:     tx,
		FunctionABI:     &functionABI,
		ContractAddress: contractAddress,
		DomainConfig:    domainConfig,
		Params:          params,
	}, handler, nil
}

func (n *Noto) recoverSignature(ctx context.Context, payload ethtypes.HexBytes0xPrefix, signature []byte) (*ethtypes.Address0xHex, error) {
	sig, err := secp256k1.DecodeCompactRSV(ctx, signature)
	if err != nil {
		return nil, err
	}
	return sig.RecoverDirect(payload, n.chainID)
}

func (n *Noto) parseCoinList(ctx context.Context, label string, states []*prototk.EndorsableState) ([]*types.NotoCoin, []*prototk.StateRef, *big.Int, error) {
	var err error
	statesUsed := make(map[string]bool)
	coins := make([]*types.NotoCoin, len(states))
	refs := make([]*prototk.StateRef, len(states))
	total := big.NewInt(0)
	for i, state := range states {
		if state.SchemaId != n.coinSchema.Id {
			return nil, nil, nil, i18n.NewError(ctx, msgs.MsgUnknownSchema, state.SchemaId)
		}
		if statesUsed[state.Id] {
			return nil, nil, nil, i18n.NewError(ctx, msgs.MsgDuplicateStateInList, label, i, state.Id)
		}
		statesUsed[state.Id] = true
		if coins[i], err = n.unmarshalCoin(state.StateDataJson); err != nil {
			return nil, nil, nil, i18n.NewError(ctx, msgs.MsgInvalidListInput, label, i, state.Id, err)
		}
		refs[i] = &prototk.StateRef{
			SchemaId: state.SchemaId,
			Id:       state.Id,
		}
		total = total.Add(total, coins[i].Amount.Int())
	}
	return coins, refs, total, nil
}

func (n *Noto) gatherCoins(ctx context.Context, inputs, outputs []*prototk.EndorsableState) (*gatheredCoins, error) {
	inCoins, inStates, inTotal, err := n.parseCoinList(ctx, "input", inputs)
	if err != nil {
		return nil, err
	}
	outCoins, outStates, outTotal, err := n.parseCoinList(ctx, "output", outputs)
	if err != nil {
		return nil, err
	}
	return &gatheredCoins{
		inCoins:   inCoins,
		inStates:  inStates,
		inTotal:   inTotal,
		outCoins:  outCoins,
		outStates: outStates,
		outTotal:  outTotal,
	}, nil
}

func (n *Noto) encodeTransactionData(ctx context.Context, transaction *prototk.TransactionSpecification) (tktypes.HexBytes, error) {
	txID, err := tktypes.ParseHexBytes(ctx, transaction.TransactionId)
	if err != nil {
		return nil, err
	}

	var data []byte
	data = append(data, types.NotoTransactionData_V0...)
	data = append(data, txID...)
	return data, nil
}

func (n *Noto) decodeTransactionData(data tktypes.HexBytes) (txID tktypes.HexBytes) {
	if len(data) < 4 {
		return nil
	}
	dataPrefix := data[0:4]
	if dataPrefix.String() != types.NotoTransactionData_V0.String() {
		return nil
	}
	return data[4:]
}

func (n *Noto) parseStatesFromEvent(txID tktypes.HexBytes, states []tktypes.Bytes32) []*prototk.StateUpdate {
	refs := make([]*prototk.StateUpdate, len(states))
	for i, state := range states {
		refs[i] = &prototk.StateUpdate{
			Id:            state.String(),
			TransactionId: txID.String(),
		}
	}
	return refs
}

func (n *Noto) HandleEventBatch(ctx context.Context, req *prototk.HandleEventBatchRequest) (*prototk.HandleEventBatchResponse, error) {
	var res prototk.HandleEventBatchResponse
	for _, ev := range req.Events {
		switch ev.SoliditySignature {
		case n.transferSignature:
			var transfer NotoTransfer_Event
			if err := json.Unmarshal([]byte(ev.DataJson), &transfer); err == nil {
				txID := n.decodeTransactionData(transfer.Data)
				res.TransactionsComplete = append(res.TransactionsComplete, &prototk.CompletedTransaction{
					TransactionId: txID.String(),
					Location:      ev.Location,
				})
				res.SpentStates = append(res.SpentStates, n.parseStatesFromEvent(txID, transfer.Inputs)...)
				res.ConfirmedStates = append(res.ConfirmedStates, n.parseStatesFromEvent(txID, transfer.Outputs)...)
			}

		case n.approvedSignature:
			var approved NotoApproved_Event
			if err := json.Unmarshal([]byte(ev.DataJson), &approved); err == nil {
				txID := n.decodeTransactionData(approved.Data)
				res.TransactionsComplete = append(res.TransactionsComplete, &prototk.CompletedTransaction{
					TransactionId: txID.String(),
					Location:      ev.Location,
				})
			}
		}
	}
	return &res, nil
}

func (n *Noto) Sign(ctx context.Context, req *prototk.SignRequest) (*prototk.SignResponse, error) {
	return nil, i18n.NewError(ctx, msgs.MsgNotImplemented)
}

func (n *Noto) GetVerifier(ctx context.Context, req *prototk.GetVerifierRequest) (*prototk.GetVerifierResponse, error) {
	return nil, i18n.NewError(ctx, msgs.MsgNotImplemented)
}

func (n *Noto) ValidateStateHashes(ctx context.Context, req *prototk.ValidateStateHashesRequest) (*prototk.ValidateStateHashesResponse, error) {
	return nil, i18n.NewError(ctx, msgs.MsgNotImplemented)
}

func (n *Noto) InitCall(ctx context.Context, req *prototk.InitCallRequest) (*prototk.InitCallResponse, error) {
	return nil, i18n.NewError(ctx, msgs.MsgNotImplemented)
}

func (n *Noto) ExecCall(ctx context.Context, req *prototk.ExecCallRequest) (*prototk.ExecCallResponse, error) {
	return nil, i18n.NewError(ctx, msgs.MsgNotImplemented)
}

func (n *Noto) BuildReceipt(ctx context.Context, req *prototk.BuildReceiptRequest) (*prototk.BuildReceiptResponse, error) {
	// TODO: Event logs for transfers would be great for Noto
	return nil, i18n.NewError(ctx, msgs.MsgNoDomainReceipt)
}
