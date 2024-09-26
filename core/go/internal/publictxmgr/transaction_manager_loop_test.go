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

package publictxmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEnginePollingCancelledContext(t *testing.T) {
	ctx, ble, _, done := newTestPublicTxManager(t, false)
	done()

	polled, _ := ble.poll(ctx)
	assert.Equal(t, -1, polled)
}

// func TestNewEnginePollingReAddStoppedOrchestrator(t *testing.T) {
// 	ctx := context.Background()
// 	mockManagedTx1 := &ptxapi.PublicTx{
// 		ID: uuid.New(),
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(1),
// 		},
// 	}
// 	mockManagedTx2 := &ptxapi.PublicTx{
// 		ID: uuid.New(),
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(2),
// 		},
// 	}

// 	ble, _ := NewTestPublicTxManager(t)

// 	ble.gasPriceClient = NewTestFixedPriceGasPriceClient(t)
// 	mTS := componentmocks.NewPublicTransactionStore(t)
// 	mBI := componentmocks.NewBlockIndexer(t)
// 	mEN := componentmocks.NewPublicTxEventNotifier(t)

// 	mEC := componentmocks.NewEthClient(t)
// 	mKM := componentmocks.NewKeyManager(t)
// 	ble.Init(ctx, mEC, mKM, mTS, mEN, mBI)
// 	ble.maxInFlightOrchestrators = 1
// 	ble.enginePollingInterval = 1 * time.Hour

// 	zeroTransactionListedForIdle := make(chan struct{})

// 	// already has a running orchestrator for the address so no new orchestrator should be started
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{mockManagedTx1, mockManagedTx2}, nil).Once()
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{}, nil).Run(func(args mock.Arguments) {
// 		close(zeroTransactionListedForIdle)
// 	}).Once()

// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{}, nil).Maybe()
// 	ble.InFlightOrchestrators = map[string]*orchestrator{
// 		testMainSigningAddress: {state: OrchestratorStateStopped, stateEntryTime: time.Now()}, // already has an orchestrator for 0x1
// 	}
// 	ble.ctx = ctx
// 	ble.poll(ctx)
// 	to := ble.InFlightOrchestrators[testMainSigningAddress]
// 	assert.NotNil(t, to)
// 	to.maxInFlightTxs = 1
// 	to.poll(ctx)
// 	<-zeroTransactionListedForIdle
// 	assert.Equal(t, OrchestratorStateIdle, to.state)
// }

// func TestNewEnginePollingStoppingAnOrchestratorAndSelf(t *testing.T) {
// 	ctx, cancelCtx := context.WithCancel(context.Background())
// 	mockManagedTx1 := &ptxapi.PublicTx{
// 		ID: uuid.New(),
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(1),
// 		},
// 	}
// 	mockManagedTx2 := &ptxapi.PublicTx{
// 		ID: uuid.New(),
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(2),
// 		},
// 	}
// 	ble, _ := NewTestPublicTxManager(t)

// 	ble.gasPriceClient = NewTestFixedPriceGasPriceClient(t)
// 	mTS := componentmocks.NewPublicTransactionStore(t)
// 	mBI := componentmocks.NewBlockIndexer(t)
// 	mEN := componentmocks.NewPublicTxEventNotifier(t)

// 	mEC := componentmocks.NewEthClient(t)
// 	mKM := componentmocks.NewKeyManager(t)
// 	ble.Init(ctx, mEC, mKM, mTS, mEN, mBI)
// 	ble.maxInFlightOrchestrators = 2
// 	ble.ctx = ctx
// 	ble.enginePollingInterval = 1 * time.Hour
// 	ble.engineLoopDone = make(chan struct{})
// 	// already has a running orchestrator for the address so no new orchestrator should be started
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{mockManagedTx1, mockManagedTx2}, nil).Once()
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{}, nil).Maybe()
// 	go ble.engineLoop()
// 	existingOrchestrator := &orchestrator{
// 		pubTxManager:                ble,
// 		orchestratorPollingInterval: ble.enginePollingInterval,
// 		state:                       OrchestratorStateIdle,
// 		stateEntryTime:              time.Now().Add(-ble.maxOrchestratorIdle).Add(-1 * time.Minute),
// 		InFlightTxsStale:            make(chan bool, 1),
// 		stopProcess:                 make(chan bool, 1),
// 		txStore:                     mTS,
// 		ethClient:                   mEC,
// 		publicTXEventNotifier:       mEN,
// 		bIndexer:                    mBI,
// 		maxInFlightTxs:              0,
// 	}
// 	ble.InFlightOrchestrators = map[string]*orchestrator{
// 		testMainSigningAddress: existingOrchestrator, // already has an orchestrator for 0x1
// 	}
// 	_, _ = existingOrchestrator.Start(ctx)
// 	ble.MarkInFlightOrchestratorsStale()
// 	<-existingOrchestrator.orchestratorLoopDone
// 	assert.Equal(t, OrchestratorStateStopped, existingOrchestrator.state)

// 	//stops OK
// 	cancelCtx()
// 	<-ble.engineLoopDone
// }

// func TestNewEnginePollingStoppingAnOrchestratorForFairnessControl(t *testing.T) {
// 	ctx := context.Background()
// 	mockManagedTx1 := &ptxapi.PublicTx{
// 		ID: uuid.New(),
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(1),
// 		},
// 	}
// 	mockManagedTx2 := &ptxapi.PublicTx{
// 		ID: uuid.New(),
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(2),
// 		},
// 	}

// 	ble, _ := NewTestPublicTxManager(t)

// 	ble.gasPriceClient = NewTestFixedPriceGasPriceClient(t)
// 	mTS := componentmocks.NewPublicTransactionStore(t)
// 	mBI := componentmocks.NewBlockIndexer(t)
// 	mEN := componentmocks.NewPublicTxEventNotifier(t)

// 	mEC := componentmocks.NewEthClient(t)
// 	mKM := componentmocks.NewKeyManager(t)
// 	ble.Init(ctx, mEC, mKM, mTS, mEN, mBI)
// 	ble.maxInFlightOrchestrators = 1
// 	ble.ctx = ctx
// 	ble.enginePollingInterval = 1 * time.Hour
// 	ble.engineLoopDone = make(chan struct{})
// 	ble.maxInFlightOrchestrators = 1
// 	existingOrchestrator := &orchestrator{
// 		orchestratorBirthTime:       time.Now().Add(-1 * time.Hour),
// 		pubTxManager:                ble,
// 		orchestratorPollingInterval: ble.enginePollingInterval,
// 		state:                       OrchestratorStateRunning,
// 		stateEntryTime:              time.Now().Add(-ble.maxOrchestratorIdle).Add(-1 * time.Minute),
// 		InFlightTxsStale:            make(chan bool, 1),
// 		stopProcess:                 make(chan bool, 1),
// 		txStore:                     mTS,
// 		ethClient:                   mEC,
// 		publicTXEventNotifier:       mEN,
// 		bIndexer:                    mBI,
// 	}
// 	// already has a running orchestrator for the address so no new orchestrator should be started
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{mockManagedTx1, mockManagedTx2}, nil).Maybe()
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{}, nil).Maybe()
// 	ble.InFlightOrchestrators = map[string]*orchestrator{
// 		testMainSigningAddress: existingOrchestrator, // already has an orchestrator for 0x1
// 	}
// 	ble.ctx = ctx
// 	ble.poll(ctx)
// 	existingOrchestrator.orchestratorLoopDone = make(chan struct{})
// 	existingOrchestrator.orchestratorLoop()
// 	<-existingOrchestrator.orchestratorLoopDone
// 	assert.Equal(t, OrchestratorStateStopped, existingOrchestrator.state)
// }

// func TestNewEnginePollingExcludePausedOrchestrator(t *testing.T) {
// 	ctx := context.Background()
// 	ble, _ := NewTestPublicTxManager(t)

// 	ble.gasPriceClient = NewTestFixedPriceGasPriceClient(t)
// 	mTS := componentmocks.NewPublicTransactionStore(t)
// 	mBI := componentmocks.NewBlockIndexer(t)
// 	mBI.On("RegisterIndexedTransactionHandler", ctx, mock.Anything).Return(nil).Once()
// 	mEN := componentmocks.NewPublicTxEventNotifier(t)

// 	mEC := componentmocks.NewEthClient(t)
// 	mKM := componentmocks.NewKeyManager(t)
// 	ble.Init(ctx, mEC, mKM, mTS, mEN, mBI)
// 	ble.maxInFlightOrchestrators = 1
// 	ble.enginePollingInterval = 1 * time.Hour

// 	// already has a running orchestrator for the address so no new orchestrator should be started
// 	listed := make(chan struct{})
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{}, nil).Run(func(args mock.Arguments) {
// 		close(listed)
// 	}).Once()
// 	ble.InFlightOrchestrators = map[string]*orchestrator{}
// 	ble.SigningAddressesPausedUntil = map[string]time.Time{testMainSigningAddress: time.Now().Add(1 * time.Hour)}
// 	_, _ = ble.Start(ctx)
// 	<-listed
// 	assert.Empty(t, ble.InFlightOrchestrators)
// }

// func TestNewEngineGetPendingFuelingTxs(t *testing.T) {
// 	ctx := context.Background()
// 	mockManagedTx1 := &ptxapi.PublicTx{
// 		ID: uuid.New(),
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(1),
// 		},
// 	}
// 	ble, _ := NewTestPublicTxManager(t)
// 	ble.gasPriceClient = NewTestFixedPriceGasPriceClient(t)
// 	mTS := componentmocks.NewPublicTransactionStore(t)
// 	mBI := componentmocks.NewBlockIndexer(t)
// 	mEN := componentmocks.NewPublicTxEventNotifier(t)

// 	mEC := componentmocks.NewEthClient(t)
// 	mKM := componentmocks.NewKeyManager(t)
// 	ble.Init(ctx, mEC, mKM, mTS, mEN, mBI)
// 	ble.ctx = ctx
// 	ble.enginePollingInterval = 1 * time.Hour

// 	// already has a running orchestrator for the address so no new orchestrator should be started
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{mockManagedTx1}, nil).Once()
// 	tx, err := ble.GetPendingFuelingTransaction(ctx, "0x0", testMainSigningAddress)
// 	assert.Equal(t, mockManagedTx1, tx)
// 	require.NoError(t, err)
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("List transaction errored")).Once()
// 	tx, err = ble.GetPendingFuelingTransaction(ctx, "0x0", testMainSigningAddress)
// 	assert.Nil(t, tx)
// 	assert.Error(t, err)
// 	assert.Regexp(t, "errored", err)
// }

// func TestNewEngineCheckTxCompleteness(t *testing.T) {
// 	ctx := context.Background()
// 	mockManagedTx1 := &ptxapi.PublicTx{
// 		ID:     uuid.New(),
// 		Status: PubTxStatusSucceeded,
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(0),
// 		},
// 	}

// 	ble, _ := NewTestPublicTxManager(t)
// 	ble.gasPriceClient = NewTestFixedPriceGasPriceClient(t)
// 	mTS := componentmocks.NewPublicTransactionStore(t)
// 	mBI := componentmocks.NewBlockIndexer(t)
// 	mEN := componentmocks.NewPublicTxEventNotifier(t)

// 	mEC := componentmocks.NewEthClient(t)
// 	mKM := componentmocks.NewKeyManager(t)
// 	ble.Init(ctx, mEC, mKM, mTS, mEN, mBI)
// 	ble.ctx = ctx
// 	ble.enginePollingInterval = 1 * time.Hour

// 	// when no nonce cached

// 	// return false for a transaction with nonce "0" that is still pending
// 	testTxWithZeroNonce := &ptxapi.PublicTx{
// 		Status: PubTxStatusPending,
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(0),
// 		},
// 	}
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{}, nil).Once()
// 	assert.False(t, ble.CheckTransactionCompleted(ctx, testTxWithZeroNonce))

// 	// for transactions with a non-zero nonce
// 	testTxToCheck := &ptxapi.PublicTx{
// 		Status: PubTxStatusPending,
// 		Transaction: &ethsigner.Transaction{
// 			From:  json.RawMessage(testMainSigningAddress),
// 			Nonce: tktypes.Uint64ToUint256(1),
// 		},
// 	}
// 	// return false when retrieve transactions failed
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("List transaction errored")).Once()
// 	assert.False(t, ble.CheckTransactionCompleted(ctx, testTxToCheck))
// 	// return false when no transactions retrieved
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{}, nil).Once()
// 	assert.False(t, ble.CheckTransactionCompleted(ctx, testTxToCheck))
// 	// return false when the retrieved transaction has a lower nonce
// 	mTS.On("ListTransactions", mock.Anything, mock.Anything).Return([]*ptxapi.PublicTx{mockManagedTx1}, nil).Once()
// 	assert.False(t, ble.CheckTransactionCompleted(ctx, testTxToCheck))

// 	// try to update nonce when transaction incomplete shouldn't take affect

// 	ble.updateCompletedTxNonce(testTxToCheck) // nonce stayed at 0
// 	assert.False(t, ble.CheckTransactionCompleted(ctx, testTxToCheck))

// 	// try to update the nonce with a completed transaction works
// 	testTxToCheck.Status = PubTxStatusFailed
// 	ble.updateCompletedTxNonce(testTxToCheck) // nonce stayed at 0
// 	assert.True(t, ble.CheckTransactionCompleted(ctx, testTxToCheck))
// }
