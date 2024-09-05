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

package main

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/rlp"
)

// Constants
const (
	ChainID     = 1337
	GasLimit    = 30 * 1000000
	EpochLength = 30000
	Nonce       = 0
	CancunTime  = 0
	ZeroBaseFee = true
)

// GenesisBuilder is an interface for building genesis blocks
type GenesisBuilder interface{}

// GenesisJSON represents the JSON structure of a genesis block
type GenesisJSON struct {
	Nonce      ethtypes.HexUint64        `json:"nonce"`
	Timestamp  ethtypes.HexUint64        `json:"timestamp"`
	GasLimit   ethtypes.HexUint64        `json:"gasLimit"`
	Difficulty ethtypes.HexUint64        `json:"difficulty"`
	MixHash    ethtypes.HexBytes0xPrefix `json:"mixHash"`
	Coinbase   *ethtypes.Address0xHex    `json:"coinbase"`
	Alloc      map[string]AllocEntry     `json:"alloc"`
}

// GenesisConfig represents the configuration for a genesis block
type GenesisConfig struct {
	ChainID     int64 `json:"chainId"`
	CancunTime  int64 `json:"cancunTime"`
	ZeroBaseFee bool  `json:"zeroBaseFee"`
}

// AllocEntry represents an allocation entry in the genesis block
type AllocEntry struct {
	Balance ethtypes.HexInteger `json:"balance"`
}

var _ GenesisBuilder = (*GenesisCliqueJSON)(nil)

// GenesisCliqueJSON represents the JSON structure of a Clique genesis block
type GenesisCliqueJSON struct {
	GenesisJSON `json:",inline"`
	Config      GenesisCliqueConfig `json:"config"`
	ExtraData   string              `json:"extraData"`
}

// GenesisCliqueConfig represents the configuration for a Clique genesis block
type GenesisCliqueConfig struct {
	GenesisConfig `json:",inline"`
	Clique        CliqueConfig `json:"clique"`
}

// CliqueConfig represents the Clique-specific configuration
type CliqueConfig struct {
	BlockPeriodSeconds int  `json:"blockperiodseconds"`
	EpochLength        int  `json:"epochlength"`
	CreateEmptyBlocks  bool `json:"createemptyblocks"`
}

// newGenesisCliqueJSON creates a new GenesisCliqueJSON instance
func newGenesisCliqueJSON(addresses ethtypes.Address0xHex) *GenesisCliqueJSON {
	oneEth := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	return &GenesisCliqueJSON{
		Config: GenesisCliqueConfig{
			GenesisConfig: GenesisConfig{
				ChainID:     ChainID,
				CancunTime:  CancunTime,
				ZeroBaseFee: ZeroBaseFee,
			},
			Clique: CliqueConfig{
				BlockPeriodSeconds: 1,
				CreateEmptyBlocks:  false,
				EpochLength:        EpochLength,
			},
		},
		GenesisJSON: GenesisJSON{
			Nonce:      0,
			Timestamp:  ethtypes.HexUint64(time.Now().Unix()),
			GasLimit:   GasLimit,
			Difficulty: 1,
			MixHash:    randBytes(32),
			Coinbase:   ethtypes.MustNewAddress("0x0000000000000000000000000000000000000000"),
			Alloc: map[string]AllocEntry{
				addresses.String(): {
					Balance: *ethtypes.NewHexInteger(
						new(big.Int).Mul(oneEth, big.NewInt(1000000000)),
					),
				},
			},
		},
		ExtraData: extraDataClique(addresses),
	}
}

// extraDataClique generates the extra data for a Clique genesis block
func extraDataClique(validators ...ethtypes.Address0xHex) string {
	extraData := "0x70616c6164696e00000000000000000000000000000000000000000000000000"

	for _, validator := range validators {
		extraData += validator.String()[2:]
	}

	return strings.ReplaceAll(fmt.Sprintf("%-236s", extraData), " ", "0")
}

var _ GenesisBuilder = (*GenesisQBFTJSON)(nil)

// GenesisQBFTJSON represents the JSON structure of a QBFT genesis block
type GenesisQBFTJSON struct {
	GenesisJSON `json:",inline"`
	ExtraData   ethtypes.HexBytes0xPrefix `json:"extraData"`
	Config      GenesisQBFTConfig         `json:"config"`
}

// GenesisQBFTConfig represents the configuration for a QBFT genesis block
type GenesisQBFTConfig struct {
	GenesisConfig `json:",inline"`
	QBFT          QBFTConfig `json:"qbft"`
}

// QBFTConfig represents the QBFT-specific configuration
type QBFTConfig struct {
	BlockPeriodSeconds    int `json:"blockperiodseconds"`
	EpochLength           int `json:"epochlength"`
	RequestTimeoutSeconds int `json:"requesttimeoutseconds"`
}

// newGenesisQBFTJSON creates a new GenesisQBFTJSON instance
func newGenesisQBFTJSON(addresses ethtypes.Address0xHex) *GenesisQBFTJSON {
	oneEth := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	return &GenesisQBFTJSON{
		Config: GenesisQBFTConfig{
			GenesisConfig: GenesisConfig{
				ChainID:     ChainID,
				CancunTime:  CancunTime,
				ZeroBaseFee: ZeroBaseFee,
			},
			QBFT: QBFTConfig{
				BlockPeriodSeconds:    1,
				EpochLength:           EpochLength,
				RequestTimeoutSeconds: 4,
			},
		},
		GenesisJSON: GenesisJSON{
			Nonce:      0,
			Timestamp:  ethtypes.HexUint64(time.Now().Unix()),
			GasLimit:   GasLimit,
			Difficulty: 1,
			MixHash:    randBytes(32),
			Coinbase:   ethtypes.MustNewAddress("0x0000000000000000000000000000000000000000"),
			Alloc: map[string]AllocEntry{
				addresses.String(): {
					Balance: *ethtypes.NewHexInteger(
						new(big.Int).Mul(oneEth, big.NewInt(1000000000)),
					),
				},
			},
		},
		ExtraData: extraDataQBFT(addresses),
	}
}

// extraDataQBFT generates the extra data for a QBFT genesis block
func extraDataQBFT(validators ...ethtypes.Address0xHex) []byte {
	vanity := make([]byte, 32)
	copy(vanity, ([]byte)("paladin"))
	var rlpValidators rlp.List
	for _, validator := range validators {
		rlpValidators = append(rlpValidators, rlp.WrapAddress(&validator))
	}
	extraDataRLP := rlp.List{
		// 32 bytes Vanity
		rlp.Data(vanity),
		// List<Validators>
		rlpValidators,
		// No Vote
		rlp.List{},
		// Round=Int(0)
		rlp.WrapInt(big.NewInt(0)),
		// 0 Seals
		rlp.List{},
	}
	return extraDataRLP.Encode()
}
