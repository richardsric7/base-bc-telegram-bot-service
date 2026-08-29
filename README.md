# base-bc-telegram-bot-service

Golang service that works with a Telegram bot to let you create and manage
ERC-20 tokens on the Base blockchain network, and swap assets on wallets you
control. See [initial-build-plan.md](./initial-build-plan.md) for the full architecture and design.

This is a self-custody tool: wallets are generated or imported by you and
stored encrypted at rest. Nobody is pre-approved by ID — access is gated by a
2FA-style pairing code instead (see below).

## Setup

1. Create a Telegram bot via [@BotFather](https://t.me/BotFather) and grab
   its token.
2. Copy `.env.example` to `.env` and fill it in:
   - `TELEGRAM_BOT_TOKEN`
   - `ADMIN_SETUP_CODE` — pick a random 6-digit number
   - `WALLET_ENCRYPTION_KEY` (a long random passphrase — back it up, losing
     it makes stored wallets unrecoverable)
   - Leave `CHAIN_NETWORK=sepolia` for testing; set it to `mainnet` (and fill
     in `BASE_RPC_URL`, `DEX_ROUTER_ADDRESS`, `DEX_QUOTER_ADDRESS`,
     `WETH_ADDRESS` for mainnet) only once you're ready for real funds.
3. Start the bot, then open it in Telegram and send `/start` followed by
   your `ADMIN_SETUP_CODE` (or open `https://t.me/<yourbot>?start=<code>`
   directly). Whoever does this first becomes the bot's admin — nobody else
   can claim it afterwards.
4. As admin, use `/generate_code` to create a 6-digit invite code for anyone
   else you want to give access to. They send it to the bot the same way to
   register as a regular user.

## Run locally

```sh
go run ./cmd/bot
```

## Run with Docker

```sh
docker compose up --build
```

## Regenerating contract bindings

The deployable `contracts/ERC20Token.sol` contract's Go bindings live in
`internal/contracts/erc20/erc20token.go`, generated with `solc` + `abigen`.
To regenerate after editing the contract:

```sh
npm install --no-save solc@0.8.24
go install github.com/ethereum/go-ethereum/cmd/abigen@v1.14.11
./scripts/compile.sh
```

## Commands

Send `/help` to the bot once it's running for the full command list
(wallets, token create/mint/burn/pause/ownership, swap quote/execute,
admin invite-code management). Wallet commands also report whether an
address has ever sent a transaction or holds a balance, since a freshly
generated wallet needs ETH before it can pay for its own gas.
