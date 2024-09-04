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

package io.kaleido.pente;

import io.kaleido.pente.evmrunner.EVMRunner;
import io.kaleido.pente.evmrunner.EVMVersion;
import org.apache.tuweni.bytes.Bytes;
import org.hyperledger.besu.datatypes.Address;
import org.hyperledger.besu.evm.frame.MessageFrame;
import org.hyperledger.besu.evm.internal.EvmConfiguration;
import org.junit.jupiter.api.Test;
import org.web3j.abi.TypeReference;
import org.web3j.abi.datatypes.Type;
import org.web3j.abi.datatypes.generated.Uint256;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.io.IOException;
import java.io.InputStream;
import java.util.List;
import java.util.Optional;
import java.util.Random;

import static io.kaleido.pente.TestUtils.firstNonMatch;
import static io.kaleido.pente.TestUtils.listContains;
import static org.junit.jupiter.api.Assertions.*;

public class SimpleStorageWrappedGetSetTest {

    @Test
    void runAnEVM() throws IOException {

        // Generate a shiny new EVM
        EVMVersion evmVersion = EVMVersion.Shanghai(new Random().nextLong(), EvmConfiguration.DEFAULT);
        EVMRunner evmRunner = new EVMRunner(evmVersion, address -> Optional.empty(), 0);

        // Load some bytecode for our first contract deploy
        String resourcePath = "contracts/testcontracts/SimpleStorageWrapped.sol/SimpleStorageWrapped.json";
        String hexByteCode;
        try (InputStream is = Thread.currentThread().getContextClassLoader().getResourceAsStream(resourcePath)) {
            assertNotNull(is);
            JsonNode node = new ObjectMapper().readTree(is);
            hexByteCode = node.get("bytecode").asText();
        }
        final Address smartContractAddress = EVMRunner.randomAddress();
        final Address sender = EVMRunner.randomAddress();
        MessageFrame deployFrame = evmRunner.runContractDeployment(
                sender,
                smartContractAddress,
                Bytes.fromHexString(hexByteCode),
                new Uint256(12345)
        );
        assertEquals(MessageFrame.State.COMPLETED_SUCCESS, deployFrame.getState());
        MessageFrame setFrame = evmRunner.runContractInvoke(
                sender,
                smartContractAddress,
                "set",
                new Uint256(23456)
        );
        assertEquals(MessageFrame.State.COMPLETED_SUCCESS, setFrame.getState());
        MessageFrame getFrame = evmRunner.runContractInvoke(
                sender,
                smartContractAddress,
                "get"
        );
        assertEquals(MessageFrame.State.COMPLETED_SUCCESS, getFrame.getState());
        List<Type<?>> returns = evmRunner.decodeReturn(getFrame, List.of(new TypeReference<Uint256>() {}));
        assertEquals(23456, ((Uint256)(returns.getFirst())).getValue().intValue());

        // Now there should be a third account for the secondary deployed contract
        assertEquals(3, evmRunner.getWorld().getQueriedAccounts().size());
        listContains(evmRunner.getWorld().getQueriedAccounts(), sender.toString());
        listContains(evmRunner.getWorld().getQueriedAccounts(), smartContractAddress.toString());
        Optional<Address> deployedContract = firstNonMatch(evmRunner.getWorld().getQueriedAccounts(), sender, smartContractAddress);
        assertTrue(deployedContract.isPresent());

        // Get the nonce of the first contract, as that should now be incremented as the deployer of the second
        assertEquals(1L, evmRunner.getWorld().get(smartContractAddress).getNonce());
    }
}