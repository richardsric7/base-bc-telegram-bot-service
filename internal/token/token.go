// Package token implements deploy/mint/burn/ownership/pause operations for
// ERC20Token contracts deployed through the bot, plus read-only info
// lookups for arbitrary ERC-20 addresses.
package token

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/chain"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/contracts/erc20"
)

// Service performs on-chain token operations.
type Service struct {
	chain *chain.Client
}

// New builds a token Service.
func New(c *chain.Client) *Service {
	return &Service{chain: c}
}

// Info is a read-only snapshot of an ERC-20 token's on-chain state.
type Info struct {
	Address     common.Address
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
	Owner       common.Address
	Paused      bool
}

// DeployResult carries the outcome of a successful deployment.
type DeployResult struct {
	Address common.Address
	Tx      *types.Transaction
	Receipt *types.Receipt
}

// Deploy creates a new ERC20Token contract, owned by the deployer's address
// (derived from key), and waits for it to be mined.
func (s *Service) Deploy(ctx context.Context, key *ecdsa.PrivateKey, name, symbol string, decimals uint8, initialSupply *big.Int) (*DeployResult, error) {
	opts, err := s.chain.TransactOpts(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	owner := ethcrypto.PubkeyToAddress(key.PublicKey)

	addr, tx, _, err := erc20.DeployERC20Token(opts, s.chain.Eth(), name, symbol, decimals, initialSupply, owner)
	if err != nil {
		return nil, fmt.Errorf("token: deploy: %w", err)
	}

	receipt, err := s.chain.WaitMined(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("token: wait mined: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("token: deploy transaction reverted (tx %s)", tx.Hash().Hex())
	}

	return &DeployResult{Address: addr, Tx: tx, Receipt: receipt}, nil
}

// Info reads name/symbol/decimals/totalSupply/owner/paused from a deployed
// ERC20Token contract.
func (s *Service) Info(ctx context.Context, address common.Address) (*Info, error) {
	c, err := erc20.NewERC20Token(address, s.chain.Eth())
	if err != nil {
		return nil, fmt.Errorf("token: bind contract: %w", err)
	}
	callOpts := &bind.CallOpts{Context: ctx}

	name, err := c.Name(callOpts)
	if err != nil {
		return nil, fmt.Errorf("token: read name: %w", err)
	}
	symbol, err := c.Symbol(callOpts)
	if err != nil {
		return nil, fmt.Errorf("token: read symbol: %w", err)
	}
	decimals, err := c.Decimals(callOpts)
	if err != nil {
		return nil, fmt.Errorf("token: read decimals: %w", err)
	}
	totalSupply, err := c.TotalSupply(callOpts)
	if err != nil {
		return nil, fmt.Errorf("token: read totalSupply: %w", err)
	}
	owner, err := c.Owner(callOpts)
	if err != nil {
		return nil, fmt.Errorf("token: read owner: %w", err)
	}
	paused, err := c.Paused(callOpts)
	if err != nil {
		return nil, fmt.Errorf("token: read paused: %w", err)
	}

	return &Info{
		Address:     address,
		Name:        name,
		Symbol:      symbol,
		Decimals:    decimals,
		TotalSupply: totalSupply,
		Owner:       owner,
		Paused:      paused,
	}, nil
}

// BalanceOf reads an account's balance of a given ERC20Token contract.
func (s *Service) BalanceOf(ctx context.Context, tokenAddress, account common.Address) (*big.Int, error) {
	c, err := erc20.NewERC20Token(tokenAddress, s.chain.Eth())
	if err != nil {
		return nil, fmt.Errorf("token: bind contract: %w", err)
	}
	return c.BalanceOf(&bind.CallOpts{Context: ctx}, account)
}

// Mint mints value base units of a token to `to`. Reverts on-chain unless
// key's address is the token owner.
func (s *Service) Mint(ctx context.Context, key *ecdsa.PrivateKey, tokenAddress, to common.Address, value *big.Int) (*types.Receipt, error) {
	return s.sendAndWait(ctx, tokenAddress, func(c *erc20.ERC20Token, opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.Mint(opts, to, value)
	}, key)
}

// Burn burns value base units from the caller's own balance.
func (s *Service) Burn(ctx context.Context, key *ecdsa.PrivateKey, tokenAddress common.Address, value *big.Int) (*types.Receipt, error) {
	return s.sendAndWait(ctx, tokenAddress, func(c *erc20.ERC20Token, opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.Burn(opts, value)
	}, key)
}

// TransferOwnership hands token ownership to a new address.
func (s *Service) TransferOwnership(ctx context.Context, key *ecdsa.PrivateKey, tokenAddress, newOwner common.Address) (*types.Receipt, error) {
	return s.sendAndWait(ctx, tokenAddress, func(c *erc20.ERC20Token, opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.TransferOwnership(opts, newOwner)
	}, key)
}

// Pause halts transfers on the token.
func (s *Service) Pause(ctx context.Context, key *ecdsa.PrivateKey, tokenAddress common.Address) (*types.Receipt, error) {
	return s.sendAndWait(ctx, tokenAddress, func(c *erc20.ERC20Token, opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.Pause(opts)
	}, key)
}

// Unpause resumes transfers on the token.
func (s *Service) Unpause(ctx context.Context, key *ecdsa.PrivateKey, tokenAddress common.Address) (*types.Receipt, error) {
	return s.sendAndWait(ctx, tokenAddress, func(c *erc20.ERC20Token, opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.Unpause(opts)
	}, key)
}

func (s *Service) sendAndWait(ctx context.Context, tokenAddress common.Address, call func(*erc20.ERC20Token, *bind.TransactOpts) (*types.Transaction, error), key *ecdsa.PrivateKey) (*types.Receipt, error) {
	c, err := erc20.NewERC20Token(tokenAddress, s.chain.Eth())
	if err != nil {
		return nil, fmt.Errorf("token: bind contract: %w", err)
	}
	opts, err := s.chain.TransactOpts(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	tx, err := call(c, opts)
	if err != nil {
		return nil, fmt.Errorf("token: send tx: %w", err)
	}
	receipt, err := s.chain.WaitMined(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("token: wait mined: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("token: transaction reverted (tx %s)", tx.Hash().Hex())
	}
	return receipt, nil
}
