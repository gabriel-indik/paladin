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

package pldclient

import (
	"testing"

	"github.com/kaleido-io/paladin/toolkit/pkg/algorithms"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
	"github.com/kaleido-io/paladin/toolkit/pkg/verifiers"
	"github.com/stretchr/testify/assert"
)

func TestKeyManagerFunctions(t *testing.T) {

	ctx, c, _, done := newTestClientAndServerHTTP(t)
	defer done()

	_, err := c.KeyManager().Wallets(ctx)
	assert.Regexp(t, "PD020702.*keymgr_wallets", err)

	_, err = c.KeyManager().ResolveKey(ctx, "key.name", algorithms.ECDSA_SECP256K1, verifiers.ETH_ADDRESS)
	assert.Regexp(t, "PD020702.*keymgr_resolveKey", err)

	_, err = c.KeyManager().ResolveEthAddress(ctx, "key.name")
	assert.Regexp(t, "PD020702.*keymgr_resolveEthAddress", err)

	_, err = c.KeyManager().ReverseKeyLookup(ctx, algorithms.ECDSA_SECP256K1, verifiers.ETH_ADDRESS, tktypes.RandAddress().String())
	assert.Regexp(t, "PD020702.*keymgr_reverseKeyLookup", err)
}