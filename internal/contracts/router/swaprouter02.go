// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package router

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// ISwapRouter02ExactInputSingleParams is an auto generated low-level Go binding around an user-defined struct.
type ISwapRouter02ExactInputSingleParams struct {
	TokenIn           common.Address
	TokenOut          common.Address
	Fee               *big.Int
	Recipient         common.Address
	AmountIn          *big.Int
	AmountOutMinimum  *big.Int
	SqrtPriceLimitX96 *big.Int
}

// SwapRouter02MetaData contains all meta data concerning the SwapRouter02 contract.
var SwapRouter02MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenIn\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenOut\",\"type\":\"address\"},{\"internalType\":\"uint24\",\"name\":\"fee\",\"type\":\"uint24\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amountIn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMinimum\",\"type\":\"uint256\"},{\"internalType\":\"uint160\",\"name\":\"sqrtPriceLimitX96\",\"type\":\"uint160\"}],\"internalType\":\"structISwapRouter02.ExactInputSingleParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"exactInputSingle\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amountOut\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amountMinimum\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"unwrapWETH9\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes[]\",\"name\":\"data\",\"type\":\"bytes[]\"}],\"name\":\"multicall\",\"outputs\":[{\"internalType\":\"bytes[]\",\"name\":\"results\",\"type\":\"bytes[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"refundETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// SwapRouter02ABI is the input ABI used to generate the binding from.
// Deprecated: Use SwapRouter02MetaData.ABI instead.
var SwapRouter02ABI = SwapRouter02MetaData.ABI

// SwapRouter02 is an auto generated Go binding around an Ethereum contract.
type SwapRouter02 struct {
	SwapRouter02Caller     // Read-only binding to the contract
	SwapRouter02Transactor // Write-only binding to the contract
	SwapRouter02Filterer   // Log filterer for contract events
}

// SwapRouter02Caller is an auto generated read-only Go binding around an Ethereum contract.
type SwapRouter02Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SwapRouter02Transactor is an auto generated write-only Go binding around an Ethereum contract.
type SwapRouter02Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SwapRouter02Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SwapRouter02Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SwapRouter02Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SwapRouter02Session struct {
	Contract     *SwapRouter02     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SwapRouter02CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SwapRouter02CallerSession struct {
	Contract *SwapRouter02Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// SwapRouter02TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SwapRouter02TransactorSession struct {
	Contract     *SwapRouter02Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// SwapRouter02Raw is an auto generated low-level Go binding around an Ethereum contract.
type SwapRouter02Raw struct {
	Contract *SwapRouter02 // Generic contract binding to access the raw methods on
}

// SwapRouter02CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SwapRouter02CallerRaw struct {
	Contract *SwapRouter02Caller // Generic read-only contract binding to access the raw methods on
}

// SwapRouter02TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SwapRouter02TransactorRaw struct {
	Contract *SwapRouter02Transactor // Generic write-only contract binding to access the raw methods on
}

// NewSwapRouter02 creates a new instance of SwapRouter02, bound to a specific deployed contract.
func NewSwapRouter02(address common.Address, backend bind.ContractBackend) (*SwapRouter02, error) {
	contract, err := bindSwapRouter02(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SwapRouter02{SwapRouter02Caller: SwapRouter02Caller{contract: contract}, SwapRouter02Transactor: SwapRouter02Transactor{contract: contract}, SwapRouter02Filterer: SwapRouter02Filterer{contract: contract}}, nil
}

// NewSwapRouter02Caller creates a new read-only instance of SwapRouter02, bound to a specific deployed contract.
func NewSwapRouter02Caller(address common.Address, caller bind.ContractCaller) (*SwapRouter02Caller, error) {
	contract, err := bindSwapRouter02(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SwapRouter02Caller{contract: contract}, nil
}

// NewSwapRouter02Transactor creates a new write-only instance of SwapRouter02, bound to a specific deployed contract.
func NewSwapRouter02Transactor(address common.Address, transactor bind.ContractTransactor) (*SwapRouter02Transactor, error) {
	contract, err := bindSwapRouter02(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SwapRouter02Transactor{contract: contract}, nil
}

// NewSwapRouter02Filterer creates a new log filterer instance of SwapRouter02, bound to a specific deployed contract.
func NewSwapRouter02Filterer(address common.Address, filterer bind.ContractFilterer) (*SwapRouter02Filterer, error) {
	contract, err := bindSwapRouter02(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SwapRouter02Filterer{contract: contract}, nil
}

// bindSwapRouter02 binds a generic wrapper to an already deployed contract.
func bindSwapRouter02(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SwapRouter02MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SwapRouter02 *SwapRouter02Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SwapRouter02.Contract.SwapRouter02Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SwapRouter02 *SwapRouter02Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SwapRouter02.Contract.SwapRouter02Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SwapRouter02 *SwapRouter02Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SwapRouter02.Contract.SwapRouter02Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SwapRouter02 *SwapRouter02CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SwapRouter02.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SwapRouter02 *SwapRouter02TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SwapRouter02.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SwapRouter02 *SwapRouter02TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SwapRouter02.Contract.contract.Transact(opts, method, params...)
}

// ExactInputSingle is a paid mutator transaction binding the contract method 0x04e45aaf.
//
// Solidity: function exactInputSingle((address,address,uint24,address,uint256,uint256,uint160) params) payable returns(uint256 amountOut)
func (_SwapRouter02 *SwapRouter02Transactor) ExactInputSingle(opts *bind.TransactOpts, params ISwapRouter02ExactInputSingleParams) (*types.Transaction, error) {
	return _SwapRouter02.contract.Transact(opts, "exactInputSingle", params)
}

// ExactInputSingle is a paid mutator transaction binding the contract method 0x04e45aaf.
//
// Solidity: function exactInputSingle((address,address,uint24,address,uint256,uint256,uint160) params) payable returns(uint256 amountOut)
func (_SwapRouter02 *SwapRouter02Session) ExactInputSingle(params ISwapRouter02ExactInputSingleParams) (*types.Transaction, error) {
	return _SwapRouter02.Contract.ExactInputSingle(&_SwapRouter02.TransactOpts, params)
}

// ExactInputSingle is a paid mutator transaction binding the contract method 0x04e45aaf.
//
// Solidity: function exactInputSingle((address,address,uint24,address,uint256,uint256,uint160) params) payable returns(uint256 amountOut)
func (_SwapRouter02 *SwapRouter02TransactorSession) ExactInputSingle(params ISwapRouter02ExactInputSingleParams) (*types.Transaction, error) {
	return _SwapRouter02.Contract.ExactInputSingle(&_SwapRouter02.TransactOpts, params)
}

// Multicall is a paid mutator transaction binding the contract method 0xac9650d8.
//
// Solidity: function multicall(bytes[] data) payable returns(bytes[] results)
func (_SwapRouter02 *SwapRouter02Transactor) Multicall(opts *bind.TransactOpts, data [][]byte) (*types.Transaction, error) {
	return _SwapRouter02.contract.Transact(opts, "multicall", data)
}

// Multicall is a paid mutator transaction binding the contract method 0xac9650d8.
//
// Solidity: function multicall(bytes[] data) payable returns(bytes[] results)
func (_SwapRouter02 *SwapRouter02Session) Multicall(data [][]byte) (*types.Transaction, error) {
	return _SwapRouter02.Contract.Multicall(&_SwapRouter02.TransactOpts, data)
}

// Multicall is a paid mutator transaction binding the contract method 0xac9650d8.
//
// Solidity: function multicall(bytes[] data) payable returns(bytes[] results)
func (_SwapRouter02 *SwapRouter02TransactorSession) Multicall(data [][]byte) (*types.Transaction, error) {
	return _SwapRouter02.Contract.Multicall(&_SwapRouter02.TransactOpts, data)
}

// RefundETH is a paid mutator transaction binding the contract method 0x12210e8a.
//
// Solidity: function refundETH() payable returns()
func (_SwapRouter02 *SwapRouter02Transactor) RefundETH(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SwapRouter02.contract.Transact(opts, "refundETH")
}

// RefundETH is a paid mutator transaction binding the contract method 0x12210e8a.
//
// Solidity: function refundETH() payable returns()
func (_SwapRouter02 *SwapRouter02Session) RefundETH() (*types.Transaction, error) {
	return _SwapRouter02.Contract.RefundETH(&_SwapRouter02.TransactOpts)
}

// RefundETH is a paid mutator transaction binding the contract method 0x12210e8a.
//
// Solidity: function refundETH() payable returns()
func (_SwapRouter02 *SwapRouter02TransactorSession) RefundETH() (*types.Transaction, error) {
	return _SwapRouter02.Contract.RefundETH(&_SwapRouter02.TransactOpts)
}

// UnwrapWETH9 is a paid mutator transaction binding the contract method 0x49404b7c.
//
// Solidity: function unwrapWETH9(uint256 amountMinimum, address recipient) payable returns()
func (_SwapRouter02 *SwapRouter02Transactor) UnwrapWETH9(opts *bind.TransactOpts, amountMinimum *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _SwapRouter02.contract.Transact(opts, "unwrapWETH9", amountMinimum, recipient)
}

// UnwrapWETH9 is a paid mutator transaction binding the contract method 0x49404b7c.
//
// Solidity: function unwrapWETH9(uint256 amountMinimum, address recipient) payable returns()
func (_SwapRouter02 *SwapRouter02Session) UnwrapWETH9(amountMinimum *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _SwapRouter02.Contract.UnwrapWETH9(&_SwapRouter02.TransactOpts, amountMinimum, recipient)
}

// UnwrapWETH9 is a paid mutator transaction binding the contract method 0x49404b7c.
//
// Solidity: function unwrapWETH9(uint256 amountMinimum, address recipient) payable returns()
func (_SwapRouter02 *SwapRouter02TransactorSession) UnwrapWETH9(amountMinimum *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _SwapRouter02.Contract.UnwrapWETH9(&_SwapRouter02.TransactOpts, amountMinimum, recipient)
}
