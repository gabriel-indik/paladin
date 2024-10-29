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

package txmgr

import (
	"context"

	"github.com/google/uuid"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/kaleido-io/paladin/core/internal/components"
	"github.com/kaleido-io/paladin/toolkit/pkg/pldapi"
	"github.com/kaleido-io/paladin/toolkit/pkg/query"
	"github.com/kaleido-io/paladin/toolkit/pkg/rpcserver"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
)

func (tm *txManager) buildRPCModule() {
	tm.rpcModule = rpcserver.NewRPCModule("ptx").
		Add("ptx_sendTransaction", tm.rpcSendTransaction()).
		Add("ptx_sendTransactions", tm.rpcSendTransactions()).
		Add("ptx_call", tm.rpcCall()).
		Add("ptx_getTransaction", tm.rpcGetTransaction()).
		Add("ptx_getTransactionFull", tm.rpcGetTransactionFull()).
		Add("ptx_getTransactionByIdempotencyKey", tm.rpcGetTransactionByIdempotencyKey()).
		Add("ptx_queryTransactions", tm.rpcQueryTransactions()).
		Add("ptx_queryTransactionsFull", tm.rpcQueryTransactionsFull()).
		Add("ptx_queryPendingTransactions", tm.rpcQueryPendingTransactions()).
		Add("ptx_getTransactionReceipt", tm.rpcGetTransactionReceipt()).
		Add("ptx_getTransactionReceiptFull", tm.rpcGetTransactionReceiptFull()).
		Add("ptx_getDomainReceipt", tm.rpcGetDomainReceipt()).
		Add("ptx_getStateReceipt", tm.rpcGetStateReceipt()).
		Add("ptx_queryTransactionReceipts", tm.rpcQueryTransactionReceipts()).
		Add("ptx_getTransactionDependencies", tm.rpcGetTransactionDependencies()).
		Add("ptx_queryPublicTransactions", tm.rpcQueryPublicTransactions()).
		Add("ptx_queryPendingPublicTransactions", tm.rpcQueryPendingPublicTransactions()).
		Add("ptx_getPublicTransactionByNonce", tm.rpcGetPublicTransactionByNonce()).
		Add("ptx_getPublicTransactionByHash", tm.rpcGetPublicTransactionByHash()).
		Add("ptx_storeABI", tm.rpcStoreABI()).
		Add("ptx_getStoredABI", tm.rpcGetStoredABI()).
		Add("ptx_decodeError", tm.rpcDecodeRevertError()).
		Add("ptx_queryStoredABIs", tm.rpcQueryStoredABIs()).
		Add("ptx_resolveVerifier", tm.rpcResolveVerifier())

	tm.debugRpcModule = rpcserver.NewRPCModule("debug").
		Add("debug_getTransactionStatus", tm.rpcDebugTransactionStatus())
}

func (tm *txManager) rpcSendTransaction() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		tx pldapi.TransactionInput,
	) (*uuid.UUID, error) {
		return tm.SendTransaction(ctx, &tx)
	})
}

func (tm *txManager) rpcSendTransactions() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		txs []*pldapi.TransactionInput,
	) ([]uuid.UUID, error) {
		return tm.SendTransactions(ctx, txs)
	})
}

func (tm *txManager) rpcCall() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		tx *pldapi.TransactionCall,
	) (result tktypes.RawJSON, err error) {
		err = tm.CallTransaction(ctx, &result, tx)
		return
	})
}

func (tm *txManager) rpcGetTransaction() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		id uuid.UUID,
	) (*pldapi.Transaction, error) {
		return tm.GetTransactionByID(ctx, id)
	})
}

func (tm *txManager) rpcGetTransactionFull() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		id uuid.UUID,
	) (*pldapi.TransactionFull, error) {
		return tm.GetTransactionByIDFull(ctx, id)
	})
}

func (tm *txManager) rpcGetTransactionByIdempotencyKey() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		idempotencyKey string,
	) (*pldapi.Transaction, error) {
		return tm.GetTransactionByIdempotencyKey(ctx, idempotencyKey)
	})
}

func (tm *txManager) rpcQueryTransactions() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		query query.QueryJSON,
	) ([]*pldapi.Transaction, error) {
		return tm.QueryTransactions(ctx, &query, false)
	})
}

func (tm *txManager) rpcQueryTransactionsFull() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		query query.QueryJSON,
	) ([]*pldapi.TransactionFull, error) {
		return tm.QueryTransactionsFull(ctx, &query, false)
	})
}

func (tm *txManager) rpcQueryPendingTransactions() rpcserver.RPCHandler {
	return rpcserver.RPCMethod2(func(ctx context.Context,
		query query.QueryJSON,
		full bool,
	) (any, error) {
		if full {
			return tm.QueryTransactionsFull(ctx, &query, true)
		}
		return tm.QueryTransactions(ctx, &query, true)
	})
}

func (tm *txManager) rpcGetTransactionReceipt() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		id uuid.UUID,
	) (*pldapi.TransactionReceipt, error) {
		return tm.GetTransactionReceiptByID(ctx, id)
	})
}

func (tm *txManager) rpcGetTransactionReceiptFull() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		id uuid.UUID,
	) (*pldapi.TransactionReceiptFull, error) {
		return tm.GetTransactionReceiptByIDFull(ctx, id)
	})
}

func (tm *txManager) rpcGetDomainReceipt() rpcserver.RPCHandler {
	return rpcserver.RPCMethod2(func(ctx context.Context,
		domain string,
		id uuid.UUID,
	) (tktypes.RawJSON, error) {
		return tm.GetDomainReceiptByID(ctx, domain, id)
	})
}

func (tm *txManager) rpcGetStateReceipt() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		id uuid.UUID,
	) (*pldapi.TransactionStates, error) {
		return tm.GetStateReceiptByID(ctx, id)
	})
}

func (tm *txManager) rpcGetTransactionDependencies() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		id uuid.UUID,
	) (*pldapi.TransactionDependencies, error) {
		return tm.GetTransactionDependencies(ctx, id)
	})
}

func (tm *txManager) rpcQueryTransactionReceipts() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		query query.QueryJSON,
	) ([]*pldapi.TransactionReceipt, error) {
		return tm.QueryTransactionReceipts(ctx, &query)
	})
}

func (tm *txManager) rpcQueryPublicTransactions() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		query query.QueryJSON,
	) ([]*pldapi.PublicTxWithBinding, error) {
		return tm.queryPublicTransactions(ctx, &query)
	})
}

func (tm *txManager) rpcQueryPendingPublicTransactions() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		query query.QueryJSON,
	) ([]*pldapi.PublicTxWithBinding, error) {
		return tm.queryPublicTransactions(ctx, query.ToBuilder().Null("transactionHash").Query())
	})
}

func (tm *txManager) rpcGetPublicTransactionByNonce() rpcserver.RPCHandler {
	return rpcserver.RPCMethod2(func(ctx context.Context,
		from tktypes.EthAddress,
		nonce tktypes.HexUint64,
	) (*pldapi.PublicTxWithBinding, error) {
		return tm.GetPublicTransactionByNonce(ctx, from, nonce)
	})
}

func (tm *txManager) rpcGetPublicTransactionByHash() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		hash tktypes.Bytes32,
	) (*pldapi.PublicTxWithBinding, error) {
		return tm.GetPublicTransactionByHash(ctx, hash)
	})
}

func (tm *txManager) rpcStoreABI() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		a abi.ABI,
	) (*tktypes.Bytes32, error) {
		return tm.storeABI(ctx, a)
	})
}

func (tm *txManager) rpcGetStoredABI() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		hash tktypes.Bytes32,
	) (*pldapi.StoredABI, error) {
		return tm.getABIByHash(ctx, hash)
	})
}

func (tm *txManager) rpcQueryStoredABIs() rpcserver.RPCHandler {
	return rpcserver.RPCMethod1(func(ctx context.Context,
		query query.QueryJSON,
	) ([]*pldapi.StoredABI, error) {
		return tm.queryABIs(ctx, &query)
	})
}

func (tm *txManager) rpcResolveVerifier() rpcserver.RPCHandler {
	return rpcserver.RPCMethod3(func(ctx context.Context,
		lookup string,
		algorithm string,
		verifierType string,
	) (string, error) {
		return tm.identityResolver.ResolveVerifier(ctx, lookup, algorithm, verifierType)
	})
}

func (tm *txManager) rpcDebugTransactionStatus() rpcserver.RPCHandler {
	return rpcserver.RPCMethod2(func(ctx context.Context,
		contractAddress string,
		id uuid.UUID,
	) (components.PrivateTxStatus, error) {
		return tm.privateTxMgr.GetTxStatus(ctx, contractAddress, id.String())
	})
}

func (tm *txManager) rpcDecodeRevertError() rpcserver.RPCHandler {
	return rpcserver.RPCMethod2(func(ctx context.Context,
		revertError tktypes.HexBytes,
		dataFormat tktypes.JSONFormatOptions,
	) (*pldapi.DecodedError, error) {
		return tm.DecodeRevertError(ctx, tm.p.DB(), revertError, dataFormat)
	})
}
