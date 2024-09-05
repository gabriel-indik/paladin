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
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
)

const (
	algorithmClique = "clique"
	algorithmQBFT   = "QBFT"
)

func main() {
	// NOTE: To get the right permissions, this needs to run inside docker against the volume of Besu
	var algo string
	var dir string

	// Parse the flags
	flag.StringVar(&dir, "dir", ".", "Caching directory")
	flag.StringVar(&algo, "algorithm", algorithmClique, fmt.Sprintf("Blockchain algorithm to use (%s/%s) - default clique", algorithmClique, algorithmQBFT))
	flag.Parse()

	dataDir := path.Join(dir, "data")
	keyFile := path.Join(dir, "key")
	keyPubFile := path.Join(dir, "key.pub")
	genesisFile := path.Join(dir, "genesis.json")

	if !fileExists(dir) {
		mkdir(dir)
	}
	if !fileExists(dataDir) {
		mkdir(dataDir)
	}

	// Check not already initialized
	if fileExists(keyFile) || fileExists(keyPubFile) || fileExists(genesisFile) {
		fmt.Println("already initialized")
		osExit(0) // this is ok - nothing to do
	}

	// Generate the key
	kp, _ := secp256k1.GenerateSecp256k1KeyPair()
	writeFileStr(keyFile, (ethtypes.HexBytes0xPrefix)(kp.PrivateKeyBytes()))
	writeFileStr(keyPubFile, (ethtypes.HexBytes0xPrefix)(kp.PublicKeyBytes()))

	var genesis GenesisBuilder
	switch algo {
	case algorithmClique:
		genesis = newGenesisCliqueJSON(kp.Address)
	case algorithmQBFT:
		genesis = newGenesisQBFTJSON(kp.Address)
	default:
		exitErrorf("unknown algorithm %q", algo)
	}
	writeFileJSON(genesisFile, &genesis)

}

var osExit = os.Exit

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	osExit(1)
}

func mkdir(dir string) {
	err := os.Mkdir(dir, 0777)
	if err != nil {
		exitErrorf("failed to make dir %q: %s", dir, err)
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func writeFileStr(filename string, stringable fmt.Stringer) {
	writeFile(filename, ([]byte)(stringable.String()))
}

func writeFileJSON(filename string, jsonable any) {
	b, err := json.MarshalIndent(jsonable, "", "  ")
	if err != nil {
		exitErrorf("failed to marshal %T: %s", jsonable, err)
	}
	writeFile(filename, b)
}

func writeFile(filename string, data []byte) {
	err := os.WriteFile(filename, data, 0666)
	if err != nil {
		exitErrorf("failed to write file %q: %s", filename, err)
	}
}

func randBytes(len int) []byte {
	b := make([]byte, len)
	_, _ = rand.Read(b)
	return b
}
