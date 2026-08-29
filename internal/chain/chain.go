// Package chain wraps an ethclient connection to a Base RPC endpoint,
// centralizing gas/nonce handling, transaction signing/broadcast, and the
// per-transaction value safety cap.
package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client wraps an ethclient.Client bound to a specific chain ID, with a
// configurable per-transaction native-value cap.
type Client struct {
	eth           *ethclient.Client
	chainID       *big.Int
	maxTxValueWei *big.Int // 0/nil disables the cap
}

// Dial connects to a Base RPC endpoint.
func Dial(rpcURL string, chainID int64, maxTxValueWei *big.Int) (*Client, error) {
	eth, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("chain: dial %s: %w", rpcURL, err)
	}
	return &Client{
		eth:           eth,
		chainID:       big.NewInt(chainID),
		maxTxValueWei: maxTxValueWei,
	}, nil
}

// ChainID returns the configured chain ID.
func (c *Client) ChainID() *big.Int { return c.chainID }

// Eth exposes the underlying ethclient for read-only calls and for building
// abigen-generated contract bindings.
func (c *Client) Eth() *ethclient.Client { return c.eth }

// ErrValueCapExceeded is returned when a transaction's native value would
// exceed the configured safety cap.
var ErrValueCapExceeded = errors.New("chain: transaction value exceeds configured MAX_TX_VALUE_ETH cap")

// checkValueCap enforces MaxTxValueWei against a transaction's native value.
func (c *Client) checkValueCap(value *big.Int) error {
	if c.maxTxValueWei == nil || c.maxTxValueWei.Sign() == 0 {
		return nil
	}
	if value != nil && value.Cmp(c.maxTxValueWei) > 0 {
		return fmt.Errorf("%w: %s wei > cap %s wei", ErrValueCapExceeded, value.String(), c.maxTxValueWei.String())
	}
	return nil
}

// TransactOpts builds a *bind.TransactOpts signer for key, ready to pass to
// an abigen contract binding. value is the native-token amount (in wei) the
// call will send, used only for the safety-cap check (0 for non-payable
// calls).
func (c *Client) TransactOpts(ctx context.Context, key *ecdsa.PrivateKey, value *big.Int) (*bind.TransactOpts, error) {
	if value == nil {
		value = big.NewInt(0)
	}
	if err := c.checkValueCap(value); err != nil {
		return nil, err
	}

	opts, err := bind.NewKeyedTransactorWithChainID(key, c.chainID)
	if err != nil {
		return nil, fmt.Errorf("chain: build transactor: %w", err)
	}
	opts.Context = ctx
	opts.Value = value
	return opts, nil
}

// BalanceOf returns the native ETH balance of an address, in wei.
func (c *Client) BalanceOf(ctx context.Context, address common.Address) (*big.Int, error) {
	return c.eth.BalanceAt(ctx, address, nil)
}

// NonceAt returns the number of transactions ever sent from address
// (its confirmed transaction count / account nonce).
func (c *Client) NonceAt(ctx context.Context, address common.Address) (uint64, error) {
	return c.eth.NonceAt(ctx, address, nil)
}

// ActivationStatus reports whether an EOA has any on-chain footprint yet.
// Base (like other EVM chains) doesn't require an explicit "activation"
// step, but a freshly generated wallet with zero balance and zero
// transactions can't pay gas for its own first transaction, so callers use
// this to warn the operator it needs funding before it can sign anything.
type ActivationStatus struct {
	Activated bool
	Nonce     uint64
	Balance   *big.Int
}

// CheckActivation reads an address's nonce and balance and reports whether
// it has ever been used on-chain (nonce > 0) or funded (balance > 0).
func (c *Client) CheckActivation(ctx context.Context, address common.Address) (*ActivationStatus, error) {
	nonce, err := c.NonceAt(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("chain: read nonce: %w", err)
	}
	balance, err := c.BalanceOf(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("chain: read balance: %w", err)
	}
	return &ActivationStatus{
		Activated: nonce > 0 || balance.Sign() > 0,
		Nonce:     nonce,
		Balance:   balance,
	}, nil
}

// SuggestGasPrice proxies ethclient's gas price oracle.
func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.eth.SuggestGasPrice(ctx)
}

// CallContract proxies ethclient for raw eth_call use (e.g. quoters).
func (c *Client) CallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	return c.eth.CallContract(ctx, msg, nil)
}

// WaitMined blocks until a transaction is mined (or ctx is done / timeout
// elapses) and returns its receipt.
func (c *Client) WaitMined(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	return bind.WaitMined(ctx, c.eth, tx)
}

// TxURL returns a Basescan explorer link for a transaction hash.
func (c *Client) TxURL(txHash string) string {
	if c.chainID.Int64() == 84532 {
		return "https://sepolia.basescan.org/tx/" + txHash
	}
	return "https://basescan.org/tx/" + txHash
}

// AddressURL returns a Basescan explorer link for an address.
func (c *Client) AddressURL(address string) string {
	if c.chainID.Int64() == 84532 {
		return "https://sepolia.basescan.org/address/" + address
	}
	return "https://basescan.org/address/" + address
}
