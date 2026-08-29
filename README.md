# base-bc-telegram-bot-service

Golang service that works with a Telegram bot to let you create and manage
ERC-20 tokens on the Base blockchain network, and swap assets on wallets you
control. See [initial-build-plan.md](./initial-build-plan.md) for the full architecture and design.

This is a self-custody tool: wallets are generated or imported by you and
stored encrypted at rest; only Telegram user IDs you list in
`ALLOWED_TELEGRAM_IDS` can operate the bot.

## Setup

1. Create a Telegram bot via [@BotFather](https://t.me/BotFather) and grab
   its token.
2. Get your numeric Telegram user ID (e.g. via [@userinfobot](https://t.me/userinfobot)).
3. Copy `.env.example` to `.env` and fill it in:
   - `TELEGRAM_BOT_TOKEN`
   - `ALLOWED_TELEGRAM_IDS`
   - `WALLET_ENCRYPTION_KEY` (a long random passphrase — back it up, losing
     it makes stored wallets unrecoverable)
   - Leave `CHAIN_NETWORK=sepolia` for testing; set it to `mainnet` (and fill
     in `BASE_RPC_URL`, `DEX_ROUTER_ADDRESS`, `DEX_QUOTER_ADDRESS`,
     `WETH_ADDRESS` for mainnet) only once you're ready for real funds.

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
(wallets, token create/mint/burn/pause/ownership, swap quote/execute).
