#!/usr/bin/env bash
# Regenerates Go bindings for the deployable ERC20Token contract.
# Requires: node (with `solc` installed via `npm install --no-save solc@0.8.24`
# in the repo root) and `abigen` (go install github.com/ethereum/go-ethereum/cmd/abigen@v1.14.11).
set -euo pipefail
cd "$(dirname "$0")/.."

node scripts/compile.js

abigen \
  --abi internal/contracts/erc20/build/ERC20Token.abi \
  --bin internal/contracts/erc20/build/ERC20Token.bin \
  --pkg erc20 \
  --type ERC20Token \
  --out internal/contracts/erc20/erc20token.go

echo "Done. Router ABI bindings under internal/contracts/router are hand-maintained"
echo "(they bind to already-deployed Uniswap V3 SwapRouter02/QuoterV2 contracts on Base),"
echo "regenerate with abigen directly against internal/contracts/router/build/*.abi if changed."
