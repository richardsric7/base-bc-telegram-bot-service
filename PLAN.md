# Base Blockchain Telegram Bot Service — Implementation Plan

## 1. Purpose

A Golang backend service, controlled through a Telegram bot, that lets an
authenticated operator:

1. **Create ERC-20 tokens** on the Base network (mainnet + Sepolia testnet).
2. **Manage tokens** they deployed (mint, burn, transfer ownership, pause,
   view supply/holders/balances).
3. **Manage wallets** — import/generate wallets, check balances, and use them
   as the signer for on-chain actions.
4. **Exchange assets** (swap ERC-20 <-> ERC-20, ETH <-> ERC-20) on Base using
   an on-chain DEX router (Uniswap V3 style, e.g. Aerodrome/Uniswap on Base),
   using a wallet the user has provided.

This is a self-custody tool: the service holds encrypted private keys on
behalf of the operator who explicitly imports/generates them and only acts on
their own explicit Telegram commands. It is not a public multi-tenant
exchange.

## 2. Tech Stack

- **Language**: Go 1.24
- **Telegram**: `go-telegram-bot-api/telegram-bot-api/v5` (long-polling)
- **Ethereum/Base client**: `go-ethereum` (`ethclient`, `accounts/abi/bind`)
- **Contracts**: Solidity `ERC20Token.sol` (OpenZeppelin-based, Ownable +
  Mintable + Burnable + Pausable), compiled with `solc` and bound with
  `abigen` into Go bindings checked into `internal/contracts/erc20`.
- **DEX integration**: Uniswap V2/V3-compatible router ABI bindings (works
  for Base's Uniswap V3 deployment and Aerodrome, both expose a router with
  `exactInputSingle`/`swapExactTokensForTokens`-style methods).
- **Storage**: SQLite via `modernc.org/sqlite` + `gorm` (zero external deps,
  easy to swap for Postgres later) — stores users, wallets (encrypted),
  tokens, and transaction history.
- **Key encryption**: AES-256-GCM, key derived from a master passphrase
  (`WALLET_ENCRYPTION_KEY` env var) with `scrypt`; wallets can also be
  imported as a geth-native encrypted keystore JSON.
- **Config**: environment variables via `.env` (`joho/godotenv`).
- **Containerization**: `Dockerfile` + `docker-compose.yml` (bot + volume for
  the SQLite file).

## 3. Architecture / Package Layout

```
cmd/bot/main.go              # entrypoint: load config, init DB, chain client, bot, wire handlers

internal/
  config/                    # env parsing & validation
  storage/                   # gorm models + repositories (users, wallets, tokens, txs)
  crypto/                    # AES-GCM encryption helpers for private keys
  chain/                     # ethclient wrapper: RPC connection, gas estimation, tx sending/waiting
  wallet/                    # wallet service: create, import, list, balance, export (guarded)
  contracts/
    erc20/                   # solidity source + go-ethereum abigen bindings for the token contract
    router/                  # minimal ABI bindings for the DEX router (swap) + ERC20 (approve/allowance)
  token/                     # token service: deploy, mint, burn, transfer ownership, pause, info
  swap/                      # swap service: quote, approve, execute swap through router
  telegram/
    bot.go                   # bot init, update loop, auth middleware
    commands/                # one file per command group (wallet.go, token.go, swap.go, help.go)
    session.go               # per-chat conversation state (multi-step flows like "create token")
    format.go                # message formatting helpers

contracts/
  ERC20Token.sol             # source of truth for the deployable token contract

scripts/
  compile.sh                 # solc + abigen regeneration script

migrations/ (embedded via gorm AutoMigrate, no separate SQL files needed)

.env.example
Dockerfile
docker-compose.yml
```

## 4. Data Model (storage)

- `User` — Telegram user ID, username, role (owner/admin), created_at.
- `Wallet` — owner user ID, label, address, encrypted private key blob, is
  default, created_at.
- `Token` — owner user ID, wallet address that deployed it, chain ID,
  contract address, name, symbol, decimals, initial supply, tx hash.
- `Transaction` — generic log of on-chain actions (deploy/mint/burn/swap),
  wallet, tx hash, status, created_at.

## 5. Telegram Command Surface

Auth: every command checks the sender's Telegram user ID against an
allow-list (`ALLOWED_TELEGRAM_IDS` env var) — unauthorized users get a
rejection message and nothing else runs.

- `/start`, `/help`
- **Wallets**
  - `/wallet_new <label>` — generate a new wallet, store encrypted, show address.
  - `/wallet_import <label>` — bot replies asking for the private key in a
    DM, deletes the message immediately after reading it.
  - `/wallet_list` — list labels + addresses (never prints private keys).
  - `/wallet_balance <label>` — ETH + known token balances.
- **Tokens**
  - `/token_create` — guided multi-step flow (name, symbol, decimals,
    initial supply, which wallet deploys/owns it) → deploys `ERC20Token.sol`.
  - `/token_list` — tokens the user has deployed.
  - `/token_info <address>` — name/symbol/decimals/totalSupply/owner.
  - `/token_mint <address> <to> <amount>`
  - `/token_burn <walletLabel> <address> <amount>`
  - `/token_transfer_owner <address> <newOwner>`
  - `/token_pause <address>` / `/token_unpause <address>`
- **Swaps**
  - `/swap_quote <fromToken|ETH> <toToken|ETH> <amount>` — read-only quote.
  - `/swap_execute <wallet> <fromToken|ETH> <toToken|ETH> <amount> <slippageBps>`
    — approves router if needed, executes swap, reports tx hash + result.
- **Utility**
  - `/tx <hash>` — status/receipt lookup.
  - `/network` — current chain (mainnet/sepolia), RPC health, gas price.

All state-changing commands show a confirmation step (inline keyboard
Confirm/Cancel) before broadcasting a transaction, and reply with a Basescan
link once mined.

## 6. Security Considerations

- Private keys are **never** logged or echoed back in full; only addresses.
- Encryption key for wallets is separate from the bot token and required at
  startup; service refuses to start without it.
- Allow-list based authorization; optional per-command role (owner-only for
  wallet import/export).
- Configurable per-tx value cap and daily spend cap (env-configurable) as a
  safety rail against fat-fingered commands or a compromised bot token.
- All outbound RPC calls go through a configured Base RPC URL (default
  public endpoint, overridable with a private/Alchemy/Infura-style URL).
- Testnet (Base Sepolia) is the default network; mainnet requires an
  explicit `CHAIN_NETWORK=mainnet` config flag, reducing the chance of an
  accidental mainnet deployment.

## 7. Implementation Phases

1. **Scaffold** — go.mod, config, storage models + migrations, Docker setup.
2. **Chain layer** — `chain` package wrapping `ethclient`, gas/nonce
   management, tx wait-for-receipt helper.
3. **Wallet management** — `crypto` + `wallet` packages, Telegram wallet
   commands.
4. **Token contract + service** — Solidity contract, compiled bindings,
   `token` package (deploy/mint/burn/pause/ownership), Telegram token
   commands including the multi-step create flow.
5. **Swap integration** — router bindings, `swap` package (quote + execute
   with approve-if-needed), Telegram swap commands.
6. **Telegram bot wiring** — command router, auth middleware, session state
   for multi-step flows, confirmation keyboards.
7. **Polish** — `/help`, error formatting, Basescan links, README usage docs.

## 8. Configuration (.env)

```
TELEGRAM_BOT_TOKEN=
ALLOWED_TELEGRAM_IDS=123456789,987654321
BASE_RPC_URL=https://sepolia.base.org
CHAIN_NETWORK=sepolia            # sepolia | mainnet
WALLET_ENCRYPTION_KEY=            # 32+ char passphrase, required
DEX_ROUTER_ADDRESS=               # router contract address for swaps
DATABASE_PATH=./data/bot.db
MAX_TX_VALUE_ETH=1.0              # safety cap, 0 = disabled
```

## 9. Out of Scope (for this pass)

- Multi-tenant custody with per-user hosted key management/HSM.
- Fiat on/off ramps.
- Liquidity provisioning / LP position management (only swaps).
- Web dashboard (Telegram is the only interface for now).
