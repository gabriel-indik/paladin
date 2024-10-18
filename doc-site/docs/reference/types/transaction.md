---
title: Transaction
---
{% include-markdown "./_includes/transaction_description.md" %}

### Example

```json
{}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | Server-generated UUID for this transaction (query only) | [`UUID`](simpletypes.md#uuid) |
| `created` | Server-generated creation timestamp for this transaction (query only) | [`Timestamp`](simpletypes.md#timestamp) |
| `idempotencyKey` | Externally supplied unique identifier for this transaction. 409 Conflict will be returned on attempt to re-submit | `string` |
| `type` | Type of transaction (public or private) | `Enum[github.com/kaleido-io/paladin/toolkit/pkg/pldapi.TransactionType]` |
| `domain` | Name of a domain - only required on input for private deploy transactions | `string` |
| `function` | Function signature - inferred from definition if not supplied | `string` |
| `abiReference` | Calculated ABI reference - required with ABI on input if not constructor | [`Bytes32`](simpletypes.md#bytes32) |
| `from` | Locator for a local signing identity to use for submission of this transaction | `string` |
| `to` | Target contract address, or null for a deploy | [`EthAddress`](simpletypes.md#ethaddress) |
| `data` | Pre-encoded array with/without function selector, array, or object input | `uint8[]` |
| `gas` | The gas limit for the transaction (optional) | [`HexUint64`](simpletypes.md#hexuint64) |
| `value` | The value transferred in the transaction (optional) | [`HexUint256`](simpletypes.md#hexuint256) |
| `maxPriorityFeePerGas` | The maximum priority fee per gas (optional) | [`HexUint256`](simpletypes.md#hexuint256) |
| `maxFeePerGas` | The maximum fee per gas (optional) | [`HexUint256`](simpletypes.md#hexuint256) |
| `gasPrice` | The gas price (optional) | [`HexUint256`](simpletypes.md#hexuint256) |
