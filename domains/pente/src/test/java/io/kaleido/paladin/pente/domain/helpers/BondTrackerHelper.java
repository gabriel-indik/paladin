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

package io.kaleido.paladin.pente.domain.helpers;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.kaleido.paladin.testbed.Testbed;
import io.kaleido.paladin.toolkit.*;

import java.io.IOException;
import java.util.HashMap;
import java.util.LinkedHashMap;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class BondTrackerHelper {
    final PenteHelper pente;
    final JsonHex.Address address;

    static final JsonABI.Parameters constructorParams = JsonABI.newParameters(
            JsonABI.newParameter("name", "string"),
            JsonABI.newParameter("symbol", "string"),
            JsonABI.newParameter("custodian", "address"),
            JsonABI.newParameter("distributionFactory", "address")
    );

    public static BondTrackerHelper deploy(PenteHelper pente, String sender, Object inputs) throws IOException {
        String bytecode = ResourceLoader.jsonResourceEntryText(
                BondTrackerHelper.class.getClassLoader(),
                "contracts/private/BondTracker.sol/BondTracker.json",
                "bytecode"
        );

        var address = pente.deploy(sender, bytecode, constructorParams, inputs);
        return new BondTrackerHelper(pente, address);
    }

    private BondTrackerHelper(PenteHelper pente, JsonHex.Address address) {
        this.pente = pente;
        this.address = address;
    }

    public JsonHex.Address address() {
        return address;
    }

    public String investorRegistry(String sender) throws IOException {
        var output = pente.call(
                "investorRegistry",
                JsonABI.newParameters(),
                JsonABI.newParameters(
                        JsonABI.newParameter("output", "address")
                ),
                sender,
                address,
                new HashMap<>()
        );
        return output.output();
    }

    public String balanceOf(String sender, String account) throws IOException {
        var output = pente.call(
                "balanceOf",
                JsonABI.newParameters(
                        JsonABI.newParameter("account", "address")
                ),
                JsonABI.newParameters(
                        JsonABI.newParameter("output", "uint256")
                ),
                sender,
                address,
                new HashMap<>() {{
                    put("account", account);
                }}
        );
        return output.output();
    }

    public void setDistribution(String sender, String addr) throws IOException {
        pente.invoke(
                "setDistribution",
                JsonABI.newParameters(
                        JsonABI.newParameter("addr", "address")
                ),
                sender,
                address,
                new HashMap<>() {{
                    put("addr", addr);
                }}
        );
    }
}
