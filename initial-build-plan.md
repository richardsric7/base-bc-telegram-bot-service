# Base Blockchain Telegram Bot Service — Implementation Plan

## 1. Purpose

A Golang backend service, controlled through a Telegram bot, that lets an
authenticated operator:

1. **Create ERC-20 tokens** on the Base network (mainnet + Sepolia testnet).
2. **Manage tokens** they deployed (mint, burn, transfer ownership, pause,
   view supply/holders/balances).
3. **Manage wallets** — import/generate wallets, check balances and on-chain
   activation status, and use them as the signer for on-chain actions.
4. **Exchange assets** (swap ERC-20 <-> ERC-20, ETH <-> ERC-20) on Base using
   an on-chain DEX router (Uniswap V3 style, e.g. Aerodrome/Uniswap on Base),
   using a wallet the user has provided.

This is a self-custody tool: the service holds encrypted private keys on
behalf of the operator who explicitly imports/generates them and only acts on
their own explicit Telegram commands. It is not a public multi-tenant
exchange. Access is not gated by a pre-configured allow-list of Telegram
IDs — instead the bot uses a 2FA-style pairing flow: an `ADMIN_SETUP_CODE`
env var claims the single admin account on first contact, and the admin can
then mint one-time 6-digit invite codes for anyone else who should have
access (see §5 and §6).

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

- `User` — Telegram user ID, username, role (`admin`/`user`), created_at.
- `InviteCode` — 6-digit code, who generated it, who redeemed it (nullable),
  expiry, created/used timestamps. One-time use, expires after 24h if unused.
- `Wallet` — owner user ID, label, address, encrypted private key blob, is
  default, created_at.
- `Token` — owner user ID, wallet address that deployed it, chain ID,
  contract address, name, symbol, decimals, initial supply, tx hash.
- `Transaction` — generic log of on-chain actions (deploy/mint/burn/swap),
  wallet, tx hash, status, created_at.

## 5. Registration / Auth (2FA-style pairing, no ID allow-list)

Nobody is pre-approved by Telegram ID. Instead:

1. **Admin bootstrap.** `ADMIN_SETUP_CODE` (a 6-digit code, env-configured)
   claims the bot's single admin account. The first Telegram user to send
   that exact code to the bot — either as the `/start` payload
   (`https://t.me/<bot>?start=123456`, Telegram's standard deep-link
   mechanism) or as a plain message — is registered as `admin`. Once an
   admin exists, the setup code no longer registers anyone (constant-time
   compared to avoid timing side-channels); further attempts are told an
   admin is already registered.
2. **Invite codes for everyone else.** The admin runs `/generate_code` to
   mint a random 6-digit `InviteCode`, valid for 24 hours and single-use.
   They share it with the new user out of band (chat, deep link, etc.). That
   user sends it to the bot the same way (`/start <code>` or a plain
   message) and is registered as `user`.
3. Any message from an unregistered Telegram ID that isn't a valid code gets
   a short instructional reply (register with the admin code if no admin
   exists yet, otherwise ask the admin for an invite code) — no command
   runs.

This replaces a static `ALLOWED_TELEGRAM_IDS` list with a pairing flow: IDs
are learned dynamically as people prove they hold a valid code, rather than
configured up front.

## 6. Telegram Command Surface

- `/start`, `/help`
- **Wallets**
  - `/wallet_new <label>` — generate a new wallet, store encrypted, show
    address and on-chain activation status.
  - `/wallet_import <label>` — bot replies asking for the private key in a
    DM, deletes the message immediately after reading it, then reports
    activation status.
  - `/wallet_list` — list labels + addresses (never prints private keys).
  - `/wallet_balance <label>` — ETH balance, transaction count, and
    activation status (see §7).
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
- **Admin** (role `admin` only)
  - `/generate_code` — mint a new 6-digit invite code (24h TTL) and an
    invite deep link.
  - `/codes` — list the admin's active, unused invite codes.
  - `/revoke_code <code>` — invalidate an unused invite code early.
- **Utility**
  - `/tx <hash>` — status/receipt lookup.
  - `/network` — current chain (mainnet/sepolia), RPC health, gas price.

All state-changing commands show a confirmation step (inline keyboard
Confirm/Cancel) before broadcasting a transaction, and reply with a Basescan
link once mined.

## 7. Wallet Activation Checks

Base is an EVM chain, so an externally-owned account (a wallet from a
private key) doesn't need an explicit "activation" transaction the way some
non-EVM chains do — but a freshly generated wallet has zero ETH and can't
pay gas for its own first transaction, which is a common source of
confusion. The bot treats an address as **activated** once it has either a
positive balance or a nonce > 0 (it has sent at least one transaction), and
surfaces that status:

- Immediately after `/wallet_new` / `/wallet_import`, so a brand-new wallet
  is clearly flagged as needing funding before use.
- On `/wallet_balance`, alongside the ETH balance and transaction count.

`internal/chain` exposes this as `Client.CheckActivation(ctx, address)`
(nonce + balance lookup); the Telegram layer formats it into a short
human-readable line.

## 8. Security Considerations

- Private keys are **never** logged or echoed back in full; only addresses.
- Encryption key for wallets is separate from the bot token and required at
  startup; service refuses to start without it.
- Registration is code-based (§5), not a static ID allow-list: the admin
  setup code is compared with a constant-time comparison, invite codes are
  single-use and time-limited, and only the `admin` role can mint new invite
  codes.
- Configurable per-tx value cap and daily spend cap (env-configurable) as a
  safety rail against fat-fingered commands or a compromised bot token.
- All outbound RPC calls go through a configured Base RPC URL (default
  public endpoint, overridable with a private/Alchemy/Infura-style URL).
- Testnet (Base Sepolia) is the default network; mainnet requires an
  explicit `CHAIN_NETWORK=mainnet` config flag, reducing the chance of an
  accidental mainnet deployment.

## 9. Implementation Phases

1. **Scaffold** — go.mod, config, storage models + migrations, Docker setup.
2. **Chain layer** — `chain` package wrapping `ethclient`, gas/nonce
   management, tx wait-for-receipt helper, nonce/activation lookups.
3. **Wallet management** — `crypto` + `wallet` packages, Telegram wallet
   commands, activation-status reporting.
4. **Token contract + service** — Solidity contract, compiled bindings,
   `token` package (deploy/mint/burn/pause/ownership), Telegram token
   commands including the multi-step create flow.
5. **Swap integration** — router bindings, `swap` package (quote + execute
   with approve-if-needed), Telegram swap commands.
6. **Registration & auth** — admin-setup-code bootstrap, invite-code
   generation/redemption, role-gated admin commands.
7. **Telegram bot wiring** — command router, registration gate, session
   state for multi-step flows, confirmation keyboards.
8. **Polish** — `/help`, error formatting, Basescan links, README usage docs.

## 10. Configuration (.env)

```
TELEGRAM_BOT_TOKEN=
ADMIN_SETUP_CODE=123456           # 6-digit code that claims the admin role once
BASE_RPC_URL=https://sepolia.base.org
CHAIN_NETWORK=sepolia            # sepolia | mainnet
WALLET_ENCRYPTION_KEY=            # 32+ char passphrase, required
DEX_ROUTER_ADDRESS=               # router contract address for swaps
DATABASE_PATH=./data/bot.db
MAX_TX_VALUE_ETH=1.0              # safety cap, 0 = disabled
```

## 11. Out of Scope (for this pass)

- Multi-tenant custody with per-user hosted key management/HSM.
- Fiat on/off ramps.
- Liquidity provisioning / LP position management (only swaps).
- Web dashboard (Telegram is the only interface for now).
