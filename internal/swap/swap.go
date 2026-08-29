// Package swap executes ERC-20/ETH swaps on Base through a Uniswap V3
// SwapRouter02-compatible router (this also covers Aerodrome/other Uniswap
// V3 forks deployed on Base, which expose the same router interface),
// quoting via the companion QuoterV2 contract.
package swap

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/chain"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/contracts/router"
)

// DefaultFeeTier is the Uniswap V3 pool fee (0.3%) used when the caller
// doesn't specify one. Expressed in hundredths of a bip (3000 = 0.3%).
const DefaultFeeTier = 3000

// Service performs swap quotes and executions.
type Service struct {
	chain         *chain.Client
	routerAddress common.Address
	quoterAddress common.Address
	wethAddress   common.Address
}

// New builds a swap Service. quoterAddress may be the zero address if quotes
// aren't needed (Quote will then return an error).
func New(c *chain.Client, routerAddress, quoterAddress, wethAddress common.Address) *Service {
	return &Service{chain: c, routerAddress: routerAddress, quoterAddress: quoterAddress, wethAddress: wethAddress}
}

// ErrQuoterNotConfigured is returned by Quote when no quoter address was set.
var ErrQuoterNotConfigured = errors.New("swap: DEX_QUOTER_ADDRESS is not configured")

// Quote reports the expected output amount for swapping amountIn of tokenIn
// for tokenOut through the configured router's pool at feeTier. Use the
// service's WETH address (Service.WETH()) as tokenIn/tokenOut to represent
// native ETH.
func (s *Service) Quote(ctx context.Context, tokenIn, tokenOut common.Address, amountIn *big.Int, feeTier uint32) (*big.Int, error) {
	if s.quoterAddress == (common.Address{}) {
		return nil, ErrQuoterNotConfigured
	}
	q, err := router.NewQuoterV2(s.quoterAddress, s.chain.Eth())
	if err != nil {
		return nil, fmt.Errorf("swap: bind quoter: %w", err)
	}
	res, err := q.QuoteExactInputSingle(&bind.CallOpts{Context: ctx}, router.IQuoterV2QuoteExactInputSingleParams{
		TokenIn:           tokenIn,
		TokenOut:          tokenOut,
		AmountIn:          amountIn,
		Fee:               new(big.Int).SetUint64(uint64(feeTier)),
		SqrtPriceLimitX96: big.NewInt(0),
	})
	if err != nil {
		return nil, fmt.Errorf("swap: quote: %w", err)
	}
	return res.AmountOut, nil
}

// Allowance reads how much of tokenAddress `owner` has approved `spender`
// (typically the router) to spend.
func (s *Service) Allowance(ctx context.Context, tokenAddress, owner common.Address) (*big.Int, error) {
	t, err := router.NewIERC20(tokenAddress, s.chain.Eth())
	if err != nil {
		return nil, fmt.Errorf("swap: bind token: %w", err)
	}
	return t.Allowance(&bind.CallOpts{Context: ctx}, owner, s.routerAddress)
}

// EnsureAllowance approves the router to spend amount of tokenAddress from
// key's wallet if the current allowance is insufficient. Returns nil if no
// approval transaction was needed.
func (s *Service) EnsureAllowance(ctx context.Context, key *ecdsa.PrivateKey, tokenAddress common.Address, amount *big.Int) (*types.Receipt, error) {
	owner := ethcrypto.PubkeyToAddress(key.PublicKey)
	current, err := s.Allowance(ctx, tokenAddress, owner)
	if err != nil {
		return nil, err
	}
	if current.Cmp(amount) >= 0 {
		return nil, nil
	}

	t, err := router.NewIERC20(tokenAddress, s.chain.Eth())
	if err != nil {
		return nil, fmt.Errorf("swap: bind token: %w", err)
	}
	opts, err := s.chain.TransactOpts(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	tx, err := t.Approve(opts, s.routerAddress, amount)
	if err != nil {
		return nil, fmt.Errorf("swap: approve: %w", err)
	}
	receipt, err := s.chain.WaitMined(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("swap: wait approve mined: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("swap: approve transaction reverted (tx %s)", tx.Hash().Hex())
	}
	return receipt, nil
}

// ExecuteParams describes a single-hop exact-input swap.
type ExecuteParams struct {
	TokenIn          common.Address // use WETH() address with Native=true for ETH input
	TokenOut         common.Address // use WETH() address with NativeOut=true for ETH output
	AmountIn         *big.Int
	AmountOutMinimum *big.Int // computed by the caller from a quote and slippage tolerance
	FeeTier          uint32   // 0 defaults to DefaultFeeTier
	Native           bool     // true if the input is native ETH (router wraps to WETH)
}

// ExecuteResult carries the outcome of a successful swap.
type ExecuteResult struct {
	AmountOut *big.Int
	Tx        *types.Transaction
	Receipt   *types.Receipt
}

// Execute runs a single-hop exact-input swap through the router. For ERC-20
// inputs, callers must call EnsureAllowance first (Execute does not do this
// automatically, so command handlers can show the user each step). For
// native ETH input, set Native=true and TokenIn to the WETH address; the
// swap's value is sent as msg.value.
func (s *Service) Execute(ctx context.Context, key *ecdsa.PrivateKey, p ExecuteParams) (*ExecuteResult, error) {
	if p.FeeTier == 0 {
		p.FeeTier = DefaultFeeTier
	}
	if p.AmountIn == nil || p.AmountIn.Sign() <= 0 {
		return nil, errors.New("swap: amountIn must be positive")
	}
	if p.AmountOutMinimum == nil || p.AmountOutMinimum.Sign() < 0 {
		return nil, errors.New("swap: amountOutMinimum must be set (use a quote minus slippage)")
	}

	r, err := router.NewSwapRouter02(s.routerAddress, s.chain.Eth())
	if err != nil {
		return nil, fmt.Errorf("swap: bind router: %w", err)
	}

	value := big.NewInt(0)
	if p.Native {
		value = p.AmountIn
	}
	opts, err := s.chain.TransactOpts(ctx, key, value)
	if err != nil {
		return nil, err
	}

	recipient := ethcrypto.PubkeyToAddress(key.PublicKey)
	tx, err := r.ExactInputSingle(opts, router.ISwapRouter02ExactInputSingleParams{
		TokenIn:           p.TokenIn,
		TokenOut:          p.TokenOut,
		Fee:               new(big.Int).SetUint64(uint64(p.FeeTier)),
		Recipient:         recipient,
		AmountIn:          p.AmountIn,
		AmountOutMinimum:  p.AmountOutMinimum,
		SqrtPriceLimitX96: big.NewInt(0),
	})
	if err != nil {
		return nil, fmt.Errorf("swap: exactInputSingle: %w", err)
	}

	receipt, err := s.chain.WaitMined(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("swap: wait mined: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("swap: transaction reverted (tx %s)", tx.Hash().Hex())
	}

	return &ExecuteResult{Tx: tx, Receipt: receipt}, nil
}

// WETH returns the configured WETH address used to represent native ETH in
// swap paths.
func (s *Service) WETH() common.Address { return s.wethAddress }

// MinAmountOut applies a slippage tolerance (in basis points, e.g. 50 = 0.5%)
// to a quoted amount, returning the minimum acceptable output.
func MinAmountOut(quoted *big.Int, slippageBps uint) *big.Int {
	if quoted == nil {
		return big.NewInt(0)
	}
	num := new(big.Int).Mul(quoted, big.NewInt(10_000-int64(slippageBps)))
	return num.Div(num, big.NewInt(10_000))
}
