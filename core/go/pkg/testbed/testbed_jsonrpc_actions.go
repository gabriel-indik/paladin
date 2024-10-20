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

package testbed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/kaleido-io/paladin/core/internal/components"
	"github.com/kaleido-io/paladin/toolkit/pkg/log"
	"github.com/kaleido-io/paladin/toolkit/pkg/pldapi"
	"github.com/kaleido-io/paladin/toolkit/pkg/prototk"
	"github.com/kaleido-io/paladin/toolkit/pkg/rpcserver"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
)

func (tb *testbed) initRPC() {
	tb.rpcModule = rpcserver.NewRPCModule("testbed").

		// Deploy a smart contract and get the deployed address
		Add("testbed_listDomains", tb.rpcListDomains()).

		// Deploy a smart contract and get the deployed address
		Add("testbed_deployBytecode", tb.rpcDeployBytecode()).

		// Performs a base ethereum transaction deploy using the
		// simple testbed transaction and key management.
		// Blocks until the Ethereum transaction is successfully confirmed
		// and returns the address emitted by the factory function
		// according to the Paladin spec.
		// Domain phases invoked prior to deploy on-chain:
		// - INIT_DEPLOY
		// - PREPARE_DEPLOY
		Add("testbed_deploy", tb.rpcTestbedDeploy()).

		// Performs a privacy preserving smart contract invoke.
		// Selecting the private states required for the transaction,
		// coordinating the required endorsements/signatures,
		// and picking a suitably anonymous signing identity to
		// submit the transaction.
		// Domain phases invoked prior to on-chain transaction:
		// - INIT_TRANSACTION     (sender node)
		// - ASSEMBLE_TRANSACTION (sender node)
		// - PREPARE_TRANSACTION  (submitter node)
		Add("testbed_invoke", tb.rpcTestbedInvoke()).

		// Prepares a privacy preserving smart contract invocation, but
		// does not actually invoke.
		// Returns an ABI encoded function call.
		Add("testbed_prepare", tb.rpcTestbedPrepare()).

		// Performs identity resolution (which in the case of the testbed is just local identities)
		Add("testbed_resolveVerifier", tb.rpcResolveVerifier())

}

func (tb *testbed) rpcListDomains() rpcserver.RPCHandler {
	return rpcserver.RPCMethod0(func(ctx context.Context) ([]string, error) {
		res := []string{}
		for name := range tb.c.DomainManager().ConfiguredDomains() {
			res = append(res, name)
		}
		return res, nil
	})
}

func (tb *testbed) rpcDeployBytecode() rpcserver.RPCHandler {
	return rpcserver.RPCMethod4(func(ctx context.Context,
		from string,
		abi abi.ABI,
		bytecode tktypes.HexBytes,
		params tktypes.RawJSON,
	) (*ethtypes.Address0xHex, error) {

		receipt, err := tb.ExecTransactionSync(ctx, &pldapi.TransactionInput{
			Transaction: pldapi.Transaction{
				Type: pldapi.TransactionTypePublic.Enum(),
				From: from,
				Data: params,
			},
			ABI:      abi,
			Bytecode: tktypes.HexBytes(bytecode),
		})
		if err != nil {
			return nil, err
		}
		return receipt.ContractAddress.Address0xHex(), nil
	})
}

func (tb *testbed) rpcTestbedDeploy() rpcserver.RPCHandler {
	return rpcserver.RPCMethod2(func(ctx context.Context,
		domainName string,
		constructorParams tktypes.RawJSON,
	) (*tktypes.EthAddress, error) {

		domain, err := tb.c.DomainManager().GetDomainByName(ctx, domainName)
		if err != nil {
			return nil, err
		}

		tx := &components.PrivateContractDeploy{
			ID:     uuid.New(),
			Domain: domain.Name(),
			Inputs: constructorParams,
		}
		err = domain.InitDeploy(ctx, tx)
		if err != nil {
			return nil, err
		}

		tx.Verifiers = make([]*prototk.ResolvedVerifier, len(tx.RequiredVerifiers))
		for i, v := range tx.RequiredVerifiers {
			resolvedKey, err := tb.ResolveKey(ctx, v.Lookup, v.Algorithm, v.VerifierType)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve key %q: %s", v.Lookup, err)
			}
			tx.Verifiers[i] = &prototk.ResolvedVerifier{
				Lookup:       v.Lookup,
				Algorithm:    v.Algorithm,
				Verifier:     resolvedKey.Verifier.Verifier,
				VerifierType: v.VerifierType,
			}
		}

		// Prepare the deployment transaction
		err = domain.PrepareDeploy(ctx, tx)
		if err != nil {
			return nil, err
		}

		// Rather than just inspecting the TX - we wait for the domain to index the event, such that
		// we know it's in the map by the time we return.
		psc, err := tb.c.DomainManager().ExecDeployAndWait(ctx, tx.ID, func() error {
			// Do the deploy - we wait for the transaction here to cover revert failures
			if tx.DeployTransaction != nil && tx.InvokeTransaction == nil {
				_, err = tb.execBaseLedgerDeployTransaction(ctx, tx.Signer, tx.DeployTransaction)
			} else if tx.InvokeTransaction != nil && tx.DeployTransaction == nil {
				_, err = tb.execBaseLedgerTransaction(ctx, tx.Signer, tx.InvokeTransaction)
			} else {
				err = fmt.Errorf("must return a transaction to invoke, or a transaction to deploy")
			}
			return err
		})
		if err != nil {
			return nil, err
		}
		addr := psc.Address()
		return &addr, nil
	})
}

func (tb *testbed) newPrivateTransaction(ctx context.Context, invocation tktypes.PrivateContractInvoke, intent prototk.TransactionSpecification_Intent) (components.DomainSmartContract, *components.PrivateTransaction, error) {
	psc, err := tb.c.DomainManager().GetSmartContractByAddress(ctx, invocation.To)
	if err != nil {
		return nil, nil, err
	}

	tx := &components.PrivateTransaction{
		ID: uuid.New(),
		Inputs: &components.TransactionInputs{
			Function: &invocation.Function,
			Domain:   psc.Domain().Name(),
			From:     invocation.From,
			To:       psc.Address(),
			Inputs:   invocation.Inputs,
			Intent:   intent,
		},
	}
	return psc, tx, err
}

func (tb *testbed) execPrivateTransaction(ctx context.Context, psc components.DomainSmartContract, tx *components.PrivateTransaction) error {

	// Testbed just uses a domain context for the duration of the TX, and flushes before returning
	dCtx := tb.c.StateManager().NewDomainContext(ctx, psc.Domain(), psc.Address(), tb.c.Persistence().DB() /* no TX */)
	defer dCtx.Close()

	// First we call init on the smart contract to:
	// - validate the transaction ABI is understood by the contract
	// - get an initial list of verifiers that need to be resolved
	if err := psc.InitTransaction(ctx, tx); err != nil {
		return err
	}

	// Gather the addresses - in the testbed we assume these all to be local
	tx.PreAssembly.Verifiers = make([]*prototk.ResolvedVerifier, len(tx.PreAssembly.RequiredVerifiers))
	for i, v := range tx.PreAssembly.RequiredVerifiers {
		resolvedKey, err := tb.ResolveKey(ctx, v.Lookup, v.Algorithm, v.VerifierType)
		if err != nil {
			return fmt.Errorf("failed to resolve key %q: %s", v.Lookup, err)
		}
		tx.PreAssembly.Verifiers[i] = &prototk.ResolvedVerifier{
			Lookup:       v.Lookup,
			Algorithm:    v.Algorithm,
			Verifier:     resolvedKey.Verifier.Verifier,
			VerifierType: v.VerifierType,
		}
	}

	// Now call assemble
	if err := psc.AssembleTransaction(dCtx, tx); err != nil {
		return err
	}

	// The testbed only handles the OK result
	switch tx.PostAssembly.AssemblyResult {
	case prototk.AssembleTransactionResponse_OK:
	default:
		return fmt.Errorf("assemble result was %s", tx.PostAssembly.AssemblyResult)
	}

	// The testbed always chooses to take the assemble output and progress to endorse
	// (no complex sequence selection routine that might result in abandonment).
	// So just write the states
	if err := psc.WritePotentialStates(dCtx, tx); err != nil {
		return err
	}

	// Gather signatures
	if err := tb.gatherSignatures(ctx, tx); err != nil {
		return err
	}

	// Gather endorsements (this would be a distributed activity across nodes in the real engine)
	if err := tb.gatherEndorsements(dCtx, psc, tx); err != nil {
		return err
	}

	log.L(ctx).Infof("Assembled and endorsed inputs=%d outputs=%d signatures=%d endorsements=%d",
		len(tx.PostAssembly.InputStates), len(tx.PostAssembly.OutputStates), len(tx.PostAssembly.Signatures), len(tx.PostAssembly.Endorsements))

	// Pick the signer for the base ledger transaction - now logically we're picking which node would do the prepare + submit phases
	if err := psc.ResolveDispatch(ctx, tx); err != nil {
		return err
	}

	// Prepare the transaction
	if err := psc.PrepareTransaction(dCtx, tx); err != nil {
		return err
	}

	// Flush the context
	if err := dCtx.FlushSync(); err != nil {
		return err
	}

	if tx.PreparedPrivateTransaction != nil && tx.PreparedPrivateTransaction.To != nil {
		nextContract, err := tb.c.DomainManager().GetSmartContractByAddress(ctx, *tx.PreparedPrivateTransaction.To)
		if err != nil {
			return err
		}
		return tb.execPrivateTransaction(ctx, nextContract, mapDirectlyToInternalPrivateTX(tx.PreparedPrivateTransaction, tx.Inputs.Intent))
	} else if tx.Inputs.Intent == prototk.TransactionSpecification_CALL {
		var ignored any
		err := tb.ExecBaseLedgerCall(ctx, &ignored, &pldapi.TransactionCall{
			TransactionInput: *tx.PreparedPublicTransaction,
			PublicCallOptions: pldapi.PublicCallOptions{
				Block: "latest",
			},
		})
		return err
	} else {
		_, err := tb.ExecTransactionSync(ctx, tx.PreparedPublicTransaction)
		return err
	}
}

func mapDirectlyToInternalPrivateTX(etx *pldapi.TransactionInput, intent prototk.TransactionSpecification_Intent) *components.PrivateTransaction {
	return &components.PrivateTransaction{
		ID: uuid.New(),
		Inputs: &components.TransactionInputs{
			Domain:   etx.Domain,
			From:     etx.From,
			To:       *etx.To,
			Function: etx.ABI[0],
			Inputs:   etx.Data,
			Intent:   intent,
		},
	}
}

func (tb *testbed) mapTransaction(tx *components.PrivateTransaction) (*tktypes.PrivateContractTransaction, error) {
	inputStates := make([]*tktypes.FullState, len(tx.PostAssembly.InputStates))
	for i, state := range tx.PostAssembly.InputStates {
		inputStates[i] = &tktypes.FullState{
			ID:     state.ID,
			Schema: state.Schema,
			Data:   []byte(state.Data),
		}
	}
	outputStates := make([]*tktypes.FullState, len(tx.PostAssembly.OutputStates))
	for i, state := range tx.PostAssembly.OutputStates {
		outputStates[i] = &tktypes.FullState{
			ID:     state.ID,
			Schema: state.Schema,
			Data:   []byte(state.Data),
		}
	}
	readStates := make([]*tktypes.FullState, len(tx.PostAssembly.ReadStates))
	for i, state := range tx.PostAssembly.ReadStates {
		readStates[i] = &tktypes.FullState{
			ID:     state.ID,
			Schema: state.Schema,
			Data:   []byte(state.Data),
		}
	}

	var functionABI *abi.Entry
	var to tktypes.EthAddress
	var paramsJSON tktypes.RawJSON
	if tx.PreparedPublicTransaction != nil {
		functionABI = tx.PreparedPublicTransaction.ABI[0]
		to = *tx.PreparedPublicTransaction.To
		paramsJSON = tx.PreparedPublicTransaction.Data
	} else {
		functionABI = tx.PreparedPrivateTransaction.ABI[0]
		to = *tx.PreparedPrivateTransaction.To
		paramsJSON = tx.PreparedPrivateTransaction.Data
	}

	return &tktypes.PrivateContractTransaction{
		FunctionABI:  functionABI,
		To:           to,
		ParamsJSON:   paramsJSON,
		InputStates:  inputStates,
		OutputStates: outputStates,
		ReadStates:   readStates,
		ExtraData:    tx.PostAssembly.ExtraData,
	}, nil
}

func (tb *testbed) rpcTestbedInvoke() rpcserver.RPCHandler {
	return rpcserver.RPCMethod2(func(ctx context.Context,
		invocation tktypes.PrivateContractInvoke,
		waitForCompletion bool,
	) (*tktypes.PrivateContractTransaction, error) {
		psc, tx, err := tb.newPrivateTransaction(ctx, invocation, prototk.TransactionSpecification_SEND_TRANSACTION)
		if err != nil {
			return nil, err
		}
		doExec := func() error {
			return tb.execPrivateTransaction(ctx, psc, tx)
		}
		if waitForCompletion {
			err = tb.c.DomainManager().ExecAndWaitTransaction(ctx, tx.ID, doExec)
		} else {
			err = doExec()
		}
		if err != nil {
			return nil, err
		}
		return tb.mapTransaction(tx)

	})
}

func (tb *testbed) rpcTestbedPrepare() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		invocation tktypes.PrivateContractInvoke,
	) (*tktypes.PrivateContractTransaction, error) {

		psc, tx, err := tb.newPrivateTransaction(ctx, invocation, prototk.TransactionSpecification_CALL)
		if err != nil {
			return nil, err
		}
		err = tb.execPrivateTransaction(ctx, psc, tx)
		if err != nil {
			return nil, err
		}
		return tb.mapTransaction(tx)
	})
}

func (tb *testbed) rpcResolveVerifier() rpcserver.RPCHandler {
	return rpcserver.RPCMethod3(func(ctx context.Context,
		lookup string,
		algorithm string,
		verifierType string,
	) (verifier string, _ error) {
		resolvedKey, err := tb.ResolveKey(ctx, lookup, algorithm, verifierType)
		if err != nil {
			return "", err
		}
		return resolvedKey.Verifier.Verifier, err
	})
}
