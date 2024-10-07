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
package domains

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/eip712"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"github.com/kaleido-io/paladin/config/pkg/confutil"
	"github.com/kaleido-io/paladin/core/pkg/blockindexer"
	"github.com/kaleido-io/paladin/core/pkg/ethclient"
	"github.com/kaleido-io/paladin/toolkit/pkg/algorithms"
	"github.com/kaleido-io/paladin/toolkit/pkg/plugintk"
	"github.com/kaleido-io/paladin/toolkit/pkg/prototk"
	"github.com/kaleido-io/paladin/toolkit/pkg/query"
	"github.com/kaleido-io/paladin/toolkit/pkg/signpayloads"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
	"github.com/kaleido-io/paladin/toolkit/pkg/verifiers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed abis/SimpleDomain.json
var simpleDomainBuild []byte // comes from Hardhat build

//go:embed abis/SimpleToken.json
var simpleTokenBuild []byte // comes from Hardhat build

func toJSONString(t *testing.T, v interface{}) string {
	b, err := json.Marshal(v)
	assert.NoError(t, err)
	return string(b)
}

type UTXOTransfer_Event struct {
	TX        tktypes.Bytes32   `json:"txId"`
	Inputs    []tktypes.Bytes32 `json:"inputs"`
	Outputs   []tktypes.Bytes32 `json:"outputs"`
	Signature tktypes.HexBytes  `json:"signature"`
}

func parseStatesFromEvent(txID tktypes.Bytes32, states []tktypes.Bytes32) []*prototk.StateUpdate {
	refs := make([]*prototk.StateUpdate, len(states))
	for i, state := range states {
		refs[i] = &prototk.StateUpdate{
			Id:            state.String(),
			TransactionId: txID.String(),
		}
	}
	return refs
}

func mustParseBuildABI(buildJSON []byte) abi.ABI {
	var buildParsed map[string]tktypes.RawJSON
	var buildABI abi.ABI
	err := json.Unmarshal(buildJSON, &buildParsed)
	if err == nil {
		err = json.Unmarshal(buildParsed["abi"], &buildABI)
	}
	if err != nil {
		panic(err)
	}
	return buildABI
}

func mustParseBuildBytecode(buildJSON []byte) tktypes.HexBytes {
	var buildParsed map[string]tktypes.RawJSON
	var byteCode tktypes.HexBytes
	err := json.Unmarshal(buildJSON, &buildParsed)
	if err == nil {
		err = json.Unmarshal(buildParsed["bytecode"], &byteCode)
	}
	if err != nil {
		panic(err)
	}
	return byteCode
}

func DeploySmartContract(t *testing.T, bi blockindexer.BlockIndexer, ecf ethclient.EthClientFactory) *tktypes.EthAddress {
	ctx := context.Background()

	simpleDomainABI := mustParseBuildABI(simpleDomainBuild)

	// In this test we deploy the factory in-line
	ec, err := ecf.HTTPClient().ABI(ctx, simpleDomainABI)
	require.NoError(t, err)

	cc, err := ec.Constructor(ctx, mustParseBuildBytecode(simpleDomainBuild))
	require.NoError(t, err)

	deployTXHash, err := cc.R(ctx).
		Signer("domain1_admin").
		Input(`{}`).
		SignAndSend()
	require.NoError(t, err)

	deployTx, err := bi.WaitForTransactionSuccess(ctx, *deployTXHash, simpleDomainABI)
	require.NoError(t, err)
	require.Equal(t, deployTx.Result.V(), blockindexer.TXResult_SUCCESS)
	return deployTx.ContractAddress
}

// Note, here we're simulating a domain that choose to support versions of a "Transfer" function
// with "string" types (rather than "address") for the from/to address and to ask Paladin to do
// verifier resolution for these. The same domain could also support "address" type inputs/outputs
// in the same ABI.
const simpleTokenTransferABI = `{
		"type": "function",
		"name": "transfer",
		"inputs": [
		  {
		    "name": "from",
			"type": "string"
		  },
		  {
		    "name": "to",
			"type": "string"
		  },
		  {
		    "name": "amount",
			"type": "uint256"
		  }
		],
		"outputs": null
	}`

func SimpleTokenTransferABI() *abi.ABI {
	return &abi.ABI{mustParseABIEntry(simpleTokenTransferABI)}
}

const simpleTokenConstructorABI = `{
		"type": "constructor",
		"inputs": [
		  {
			"name": "notary",
			"type": "string"
		  },
		  {
			"name": "name",
			"type": "string"
		  },
		  {
			"name": "symbol",
			"type": "string"
		  }
		],
		"outputs": null
	}`

func SimpleTokenConstructorABI() *abi.ABI {
	return &abi.ABI{mustParseABIEntry(simpleTokenConstructorABI)}
}

func SimpleTokenDomain(t *testing.T, ctx context.Context) plugintk.PluginBase {
	simpleDomainABI := mustParseBuildABI(simpleDomainBuild)
	simpleTokenABI := mustParseBuildABI(simpleTokenBuild)

	transferABI := simpleTokenABI.Events()["UTXOTransfer"]
	require.NotEmpty(t, transferABI)
	transferSignature := transferABI.SolString()

	simpleTokenStateSchema := `{
		"type": "tuple",
		"internalType": "struct SimpleToken",
		"components": [
			{
				"name": "salt",
				"type": "bytes32"
			},
			{
				"name": "owner",
				"type": "address",
				"indexed": true
			},
			{
				"name": "amount",
				"type": "uint256",
				"indexed": true
			}
		]
	}`

	fakeDeployPayload := `{
		"notary": "domain1.contract1.notary",
		"name": "FakeToken1",
		"symbol": "FT1"
	}`

	type fakeTransferParser struct {
		From   string               `json:"from,omitempty"`
		To     string               `json:"to,omitempty"`
		Amount *ethtypes.HexInteger `json:"amount"`
	}

	type simpleTokenParser struct {
		Salt   tktypes.HexBytes      `json:"salt"`
		Owner  ethtypes.Address0xHex `json:"owner"`
		Amount *ethtypes.HexInteger  `json:"amount"`
	}

	contractDataABI := &abi.ParameterArray{
		{Name: "notaryLocator", Type: "string"},
	}

	type simpleTokenConfigParser struct {
		NotaryLocator string `json:"notaryLocator"`
	}

	return plugintk.NewDomain(func(callbacks plugintk.DomainCallbacks) plugintk.DomainAPI {

		var simpleTokenSchemaID string
		var chainID int64
		simpleTokenSelection := func(ctx context.Context, fromAddr *ethtypes.Address0xHex, contractAddr string, amount *big.Int) ([]*simpleTokenParser, []*prototk.StateRef, *big.Int, error) {
			var lastStateTimestamp int64
			total := big.NewInt(0)
			coins := []*simpleTokenParser{}
			stateRefs := []*prototk.StateRef{}
			for {
				// Simple oldest coin first algo
				jq := &query.QueryJSON{
					Limit: confutil.P(10),
					Sort:  []string{".created"},
					Statements: query.Statements{
						Ops: query.Ops{
							Eq: []*query.OpSingleVal{
								{Op: query.Op{Field: "owner"}, Value: tktypes.JSONString(fromAddr.String())},
							},
						},
					},
				}
				if lastStateTimestamp > 0 {
					jq.GT = []*query.OpSingleVal{
						{Op: query.Op{Field: ".created"}, Value: tktypes.RawJSON(strconv.FormatInt(lastStateTimestamp, 10))},
					}
				}
				res, err := callbacks.FindAvailableStates(ctx, &prototk.FindAvailableStatesRequest{
					ContractAddress: contractAddr,
					SchemaId:        simpleTokenSchemaID,
					QueryJson:       tktypes.JSONString(jq).String(),
				})
				if err != nil {
					return nil, nil, nil, err
				}
				states := res.States
				if len(states) == 0 {
					return nil, nil, nil, fmt.Errorf("insufficient funds (available=%s)", total.Text(10))
				}
				for _, state := range states {
					lastStateTimestamp = state.StoredAt
					// Note: More sophisticated coin selection might prefer states that aren't locked to a sequence
					var coin simpleTokenParser
					if err := json.Unmarshal([]byte(state.DataJson), &coin); err != nil {
						return nil, nil, nil, fmt.Errorf("coin %s is invalid: %s", state.Id, err)
					}
					total = total.Add(total, coin.Amount.BigInt())
					stateRefs = append(stateRefs, &prototk.StateRef{
						Id:       state.Id,
						SchemaId: state.SchemaId,
					})
					coins = append(coins, &coin)
					if total.Cmp(amount) >= 0 {
						// We've got what we need - return how much over we are
						return coins, stateRefs, new(big.Int).Sub(total, amount), nil
					}
				}
			}
		}

		validateTransferTransactionInput := func(tx *prototk.TransactionSpecification) (*ethtypes.Address0xHex, string, *fakeTransferParser) {
			assert.JSONEq(t, simpleTokenTransferABI, tx.FunctionAbiJson)
			assert.Equal(t, "function transfer(string memory from, string memory to, uint256 amount) external { }", tx.FunctionSignature)
			var inputs fakeTransferParser
			err := json.Unmarshal([]byte(tx.FunctionParamsJson), &inputs)
			require.NoError(t, err)
			assert.Greater(t, inputs.Amount.BigInt().Sign(), 0)
			contractAddr, err := ethtypes.NewAddress(tx.ContractInfo.ContractAddress)
			require.NoError(t, err)
			configValues, err := contractDataABI.DecodeABIData(tx.ContractInfo.ContractConfig, 0)
			require.NoError(t, err)
			configJSON, err := tktypes.StandardABISerializer().SerializeJSON(configValues)
			require.NoError(t, err)
			var config simpleTokenConfigParser
			err = json.Unmarshal(configJSON, &config)
			require.NoError(t, err)
			assert.NotEmpty(t, config.NotaryLocator)
			return contractAddr, config.NotaryLocator, &inputs
		}

		extractTransferVerifiers := func(txSpec *prototk.TransactionSpecification, txInputs *fakeTransferParser, verifiers []*prototk.ResolvedVerifier) (senderAddr, fromAddr, toAddr *ethtypes.Address0xHex) {
			for _, v := range verifiers {
				if txSpec.From != "" && v.Lookup == txSpec.From {
					senderAddr = ethtypes.MustNewAddress(v.Verifier)
				}
				if txInputs.From != "" && v.Lookup == txInputs.From {
					fromAddr = ethtypes.MustNewAddress(v.Verifier)
				}
				if txInputs.To != "" && v.Lookup == txInputs.To {
					toAddr = ethtypes.MustNewAddress(v.Verifier)
				}
			}
			assert.True(t, txInputs.From == "" || (fromAddr != nil && *fromAddr != ethtypes.Address0xHex{}))
			assert.True(t, txInputs.To == "" || (toAddr != nil && *toAddr != ethtypes.Address0xHex{}))
			return
		}

		typedDataV4TransferWithSalts := func(contract *ethtypes.Address0xHex, inputs, outputs []*simpleTokenParser) (tktypes.HexBytes, error) {
			typeSet := eip712.TypeSet{
				"FakeTransfer": {
					{Name: "inputs", Type: "Coin[]"},
					{Name: "outputs", Type: "Coin[]"},
				},
				"Coin": {
					{Name: "salt", Type: "bytes32"},
					{Name: "owner", Type: "address"},
					{Name: "amount", Type: "uint256"},
				},
				eip712.EIP712Domain: {
					{Name: "name", Type: "string"},
					{Name: "version", Type: "string"},
					{Name: "chainId", Type: "uint256"},
					{Name: "verifyingContract", Type: "address"},
				},
			}
			messageInputs := make([]interface{}, len(inputs))
			for i, input := range inputs {
				messageInputs[i] = map[string]interface{}{
					"salt":   input.Salt.String(),
					"owner":  input.Owner.String(),
					"amount": input.Amount.String(),
				}
			}
			messageOutputs := make([]interface{}, len(outputs))
			for i, output := range outputs {
				messageOutputs[i] = map[string]interface{}{
					"salt":   output.Salt.String(),
					"owner":  output.Owner.String(),
					"amount": output.Amount.String(),
				}
			}
			tdv4, err := eip712.EncodeTypedDataV4(context.Background(), &eip712.TypedData{
				Types:       typeSet,
				PrimaryType: "FakeTransfer",
				Domain: map[string]interface{}{
					"name":              "FakeTransfer",
					"version":           "0.0.1",
					"chainId":           chainID,
					"verifyingContract": contract,
				},
				Message: map[string]interface{}{
					"inputs":  messageInputs,
					"outputs": messageOutputs,
				},
			})
			return tktypes.HexBytes(tdv4), err
		}

		return &plugintk.DomainAPIBase{Functions: &plugintk.DomainAPIFunctions{

			ConfigureDomain: func(ctx context.Context, req *prototk.ConfigureDomainRequest) (*prototk.ConfigureDomainResponse, error) {
				assert.Equal(t, "domain1", req.Name)
				assert.JSONEq(t, `{"some":"config"}`, req.ConfigJson)
				assert.Equal(t, int64(1337), req.ChainId) // from tools/besu_bootstrap
				chainID = req.ChainId

				var eventsABI abi.ABI
				eventsABI = append(eventsABI, transferABI)
				eventsJSON, err := json.Marshal(eventsABI)
				require.NoError(t, err)

				return &prototk.ConfigureDomainResponse{
					DomainConfig: &prototk.DomainConfig{
						BaseLedgerSubmitConfig: &prototk.BaseLedgerSubmitConfig{
							SubmitMode: prototk.BaseLedgerSubmitConfig_ENDORSER_SUBMISSION,
						},
						AbiStateSchemasJson: []string{simpleTokenStateSchema},
						AbiEventsJson:       string(eventsJSON),
					},
				}, nil
			},

			InitDomain: func(ctx context.Context, req *prototk.InitDomainRequest) (*prototk.InitDomainResponse, error) {
				assert.Len(t, req.AbiStateSchemas, 1)
				simpleTokenSchemaID = req.AbiStateSchemas[0].Id
				assert.Equal(t, "type=SimpleToken(bytes32 salt,address owner,uint256 amount),labels=[owner,amount]", req.AbiStateSchemas[0].Signature)
				return &prototk.InitDomainResponse{}, nil
			},

			InitDeploy: func(ctx context.Context, req *prototk.InitDeployRequest) (*prototk.InitDeployResponse, error) {
				assert.JSONEq(t, fakeDeployPayload, req.Transaction.ConstructorParamsJson)
				return &prototk.InitDeployResponse{
					RequiredVerifiers: []*prototk.ResolveVerifierRequest{
						{
							Lookup:       "domain1.contract1.notary",
							Algorithm:    algorithms.ECDSA_SECP256K1,
							VerifierType: verifiers.ETH_ADDRESS,
						},
					},
				}, nil
			},

			PrepareDeploy: func(ctx context.Context, req *prototk.PrepareDeployRequest) (*prototk.PrepareDeployResponse, error) {
				assert.JSONEq(t, `{
					"notary": "domain1.contract1.notary",
					"name": "FakeToken1",
					"symbol": "FT1"
				}`, req.Transaction.ConstructorParamsJson)
				assert.Len(t, req.ResolvedVerifiers, 1)
				assert.Equal(t, algorithms.ECDSA_SECP256K1, req.ResolvedVerifiers[0].Algorithm)
				assert.Equal(t, verifiers.ETH_ADDRESS, req.ResolvedVerifiers[0].VerifierType)
				assert.Equal(t, "domain1.contract1.notary", req.ResolvedVerifiers[0].Lookup)
				assert.NotEmpty(t, req.ResolvedVerifiers[0].Verifier)
				return &prototk.PrepareDeployResponse{
					Signer: confutil.P(fmt.Sprintf("domain1/transactions/%s", req.Transaction.TransactionId)),
					Transaction: &prototk.BaseLedgerTransaction{
						FunctionAbiJson: toJSONString(t, simpleDomainABI.Functions()["newSimpleTokenNotarized"]),
						ParamsJson: fmt.Sprintf(`{
							"txId": "%s",
							"notary": "%s",
							"notaryLocator": "domain1.contract1.notary"
						}`, req.Transaction.TransactionId, req.ResolvedVerifiers[0].Verifier),
					},
				}, nil
			},

			InitTransaction: func(ctx context.Context, req *prototk.InitTransactionRequest) (*prototk.InitTransactionResponse, error) {
				_, notaryLocator, txInputs := validateTransferTransactionInput(req.Transaction)

				// We require ethereum addresses for the "from" and "to" addresses to actually
				// execute the transaction. See notes above about this.
				requiredVerifiers := []*prototk.ResolveVerifierRequest{
					{
						Lookup:       req.Transaction.From,
						Algorithm:    algorithms.ECDSA_SECP256K1,
						VerifierType: verifiers.ETH_ADDRESS,
					},
					{
						Lookup:       notaryLocator,
						Algorithm:    algorithms.ECDSA_SECP256K1,
						VerifierType: verifiers.ETH_ADDRESS,
					},
				}
				if txInputs.From != "" {
					requiredVerifiers = append(requiredVerifiers, &prototk.ResolveVerifierRequest{
						Lookup:       txInputs.From,
						Algorithm:    algorithms.ECDSA_SECP256K1,
						VerifierType: verifiers.ETH_ADDRESS,
					})
				}
				if txInputs.To != "" && (txInputs.From == "" || txInputs.From != txInputs.To) {
					requiredVerifiers = append(requiredVerifiers, &prototk.ResolveVerifierRequest{
						Lookup:       txInputs.To,
						Algorithm:    algorithms.ECDSA_SECP256K1,
						VerifierType: verifiers.ETH_ADDRESS,
					})
				}
				return &prototk.InitTransactionResponse{
					RequiredVerifiers: requiredVerifiers,
				}, nil
			},

			AssembleTransaction: func(ctx context.Context, req *prototk.AssembleTransactionRequest) (_ *prototk.AssembleTransactionResponse, err error) {
				contractAddr, notaryLocator, txInputs := validateTransferTransactionInput(req.Transaction)
				_, fromAddr, toAddr := extractTransferVerifiers(req.Transaction, txInputs, req.ResolvedVerifiers)
				amount := txInputs.Amount.BigInt()
				toKeep := new(big.Int)
				coinsToSpend := []*simpleTokenParser{}
				stateRefsToSpend := []*prototk.StateRef{}
				if txInputs.From != "" {
					coinsToSpend, stateRefsToSpend, toKeep, err = simpleTokenSelection(ctx, fromAddr, req.Transaction.ContractInfo.ContractAddress, amount)
					if err != nil {
						return nil, err
					}
				}
				newStates := []*prototk.NewState{}
				newCoins := []*simpleTokenParser{}
				if fromAddr != nil && toKeep.Sign() > 0 {
					// Generate a state to keep for ourselves
					coin := simpleTokenParser{
						Salt:   tktypes.RandBytes(32),
						Owner:  *fromAddr,
						Amount: (*ethtypes.HexInteger)(toKeep),
					}
					newCoins = append(newCoins, &coin)

					newStates = append(newStates, &prototk.NewState{
						SchemaId:         simpleTokenSchemaID,
						StateDataJson:    toJSONString(t, &coin),
						DistributionList: []string{toAddr.String()},
					})
				}
				if toAddr != nil && amount.Sign() > 0 {
					// Generate the coin to transfer
					coin := simpleTokenParser{
						Salt:   tktypes.RandBytes(32),
						Owner:  *toAddr,
						Amount: (*ethtypes.HexInteger)(amount),
					}
					newCoins = append(newCoins, &coin)
					newStates = append(newStates, &prototk.NewState{
						SchemaId:      simpleTokenSchemaID,
						StateDataJson: toJSONString(t, &coin),
					})
				}
				eip712Payload, err := typedDataV4TransferWithSalts(contractAddr, coinsToSpend, newCoins)
				require.NoError(t, err)
				return &prototk.AssembleTransactionResponse{
					AssembledTransaction: &prototk.AssembledTransaction{
						InputStates:  stateRefsToSpend,
						OutputStates: newStates,
					},
					AssemblyResult: prototk.AssembleTransactionResponse_OK,
					AttestationPlan: []*prototk.AttestationRequest{
						{
							Name:            "sender",
							AttestationType: prototk.AttestationType_SIGN,
							Algorithm:       algorithms.ECDSA_SECP256K1,
							VerifierType:    verifiers.ETH_ADDRESS,
							PayloadType:     signpayloads.OPAQUE_TO_RSV,
							Payload:         eip712Payload,
							Parties: []string{
								req.Transaction.From,
							},
						},
						{
							Name:            "notary",
							AttestationType: prototk.AttestationType_ENDORSE,
							// we expect an endorsement is of the form ENDORSER_SUBMIT - so we need an eth signing key to exist
							Algorithm:    algorithms.ECDSA_SECP256K1,
							VerifierType: verifiers.ETH_ADDRESS,
							PayloadType:  signpayloads.OPAQUE_TO_RSV,
							Parties: []string{
								notaryLocator,
							},
						},
					},
				}, nil
			},

			EndorseTransaction: func(ctx context.Context, req *prototk.EndorseTransactionRequest) (*prototk.EndorseTransactionResponse, error) {
				contractAddr, notaryLocator, txInputs := validateTransferTransactionInput(req.Transaction)
				senderAddr, fromAddr, toAddr := extractTransferVerifiers(req.Transaction, txInputs, req.ResolvedVerifiers)
				assert.Equal(t, req.EndorsementVerifier.Lookup, req.EndorsementRequest.Parties[0])
				assert.Equal(t, req.EndorsementVerifier.Lookup, notaryLocator)

				inCoins := make([]*simpleTokenParser, len(req.Inputs))
				for i, input := range req.Inputs {
					assert.Equal(t, simpleTokenSchemaID, input.SchemaId)
					if err := json.Unmarshal([]byte(input.StateDataJson), &inCoins[i]); err != nil {
						return nil, fmt.Errorf("invalid input[%d] (%s): %s", i, input.Id, err)
					}
				}
				outCoins := make([]*simpleTokenParser, len(req.Outputs))
				for i, output := range req.Outputs {
					assert.Equal(t, simpleTokenSchemaID, output.SchemaId)
					if err := json.Unmarshal([]byte(output.StateDataJson), &outCoins[i]); err != nil {
						return nil, fmt.Errorf("invalid output[%d] (%s): %s", i, output.Id, err)
					}
				}

				// Recover the signature
				signaturePayload, err := typedDataV4TransferWithSalts(contractAddr, inCoins, outCoins)
				require.NoError(t, err)
				var signerVerification *prototk.AttestationResult
				for _, ar := range req.Signatures {
					if ar.AttestationType == prototk.AttestationType_SIGN &&
						ar.Name == "sender" &&
						ar.Verifier.Algorithm == algorithms.ECDSA_SECP256K1 &&
						ar.Verifier.VerifierType == verifiers.ETH_ADDRESS {
						signerVerification = ar
						break
					}
				}
				assert.NotNil(t, signerVerification)
				sig, err := secp256k1.DecodeCompactRSV(context.Background(), signerVerification.Payload)
				require.NoError(t, err)
				signerAddr, err := sig.RecoverDirect(signaturePayload, chainID)
				require.NoError(t, err)

				// There would need to be minting/spending rules here - we just check the signature
				assert.Equal(t, signerAddr.String(), signerVerification.Verifier.Verifier)

				// Check the math
				if fromAddr != nil && toAddr != nil {
					assert.Equal(t, senderAddr, fromAddr)
					inTotal := big.NewInt(0)
					for _, c := range inCoins {
						inTotal = inTotal.Add(inTotal, c.Amount.BigInt())
					}
					outTotal := big.NewInt(0)
					for _, c := range outCoins {
						outTotal = outTotal.Add(outTotal, c.Amount.BigInt())
					}
					assert.True(t, inTotal.Cmp(outTotal) == 0)
				} else {
					// NOTE: No minting controls in this demo example
					if fromAddr == nil {
						assert.Len(t, inCoins, 0)
					}
					if toAddr == nil {
						assert.Len(t, outCoins, 0)
					}
				}

				return &prototk.EndorseTransactionResponse{
					EndorsementResult: prototk.EndorseTransactionResponse_ENDORSER_SUBMIT,
				}, nil
			},

			PrepareTransaction: func(ctx context.Context, req *prototk.PrepareTransactionRequest) (*prototk.PrepareTransactionResponse, error) {
				var signerSignature tktypes.HexBytes
				for _, att := range req.AttestationResult {
					if att.AttestationType == prototk.AttestationType_SIGN && att.Name == "sender" {
						signerSignature = att.Payload
					}
				}
				spentStateIds := make([]string, len(req.InputStates))
				for i, s := range req.InputStates {
					spentStateIds[i] = s.Id
				}
				newStateIds := make([]string, len(req.OutputStates))
				for i, s := range req.OutputStates {
					newStateIds[i] = s.Id
				}
				return &prototk.PrepareTransactionResponse{
					Transaction: &prototk.BaseLedgerTransaction{
						FunctionAbiJson: toJSONString(t, simpleTokenABI.Functions()["executeNotarized"]),
						ParamsJson: toJSONString(t, map[string]interface{}{
							"txId":      req.Transaction.TransactionId,
							"inputs":    spentStateIds,
							"outputs":   newStateIds,
							"signature": signerSignature,
						}),
					},
				}, nil
			},

			HandleEventBatch: func(ctx context.Context, req *prototk.HandleEventBatchRequest) (*prototk.HandleEventBatchResponse, error) {
				var res prototk.HandleEventBatchResponse
				for _, ev := range req.Events {
					switch ev.SoliditySignature {
					case transferSignature:
						var transfer UTXOTransfer_Event
						if err := json.Unmarshal([]byte(ev.DataJson), &transfer); err == nil {
							res.TransactionsComplete = append(res.TransactionsComplete, &prototk.CompletedTransaction{
								TransactionId: transfer.TX.String(),
								Location:      ev.Location,
							})
							res.SpentStates = append(res.SpentStates, parseStatesFromEvent(transfer.TX, transfer.Inputs)...)
							res.ConfirmedStates = append(res.ConfirmedStates, parseStatesFromEvent(transfer.TX, transfer.Outputs)...)
						}
					}
				}
				return &res, nil
			},
		}}
	})
}

func mustParseABIEntry(abiEntryJSON string) *abi.Entry {
	var abiEntry abi.Entry
	err := json.Unmarshal([]byte(abiEntryJSON), &abiEntry)
	if err != nil {
		panic(err)
	}
	return &abiEntry
}
