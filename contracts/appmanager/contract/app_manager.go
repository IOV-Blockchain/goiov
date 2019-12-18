// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"strings"

	ethereum "github.com/CarLiveChainCo/goiov"
	"github.com/CarLiveChainCo/goiov/accounts/abi"
	"github.com/CarLiveChainCo/goiov/accounts/abi/bind"
	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/event"
)

// AppManagerABI is the input ABI used to generate the binding from.
const AppManagerABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"admin\",\"type\":\"address\"}],\"name\":\"removeAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"appId\",\"type\":\"string\"}],\"name\":\"getAppCreatorAddr\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"appId\",\"type\":\"string\"},{\"name\":\"consensusType\",\"type\":\"uint8\"}],\"name\":\"setAppConsensusType\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"appId\",\"type\":\"string\"},{\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"setAppCreatorAddr\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"appId\",\"type\":\"string\"},{\"name\":\"addrs\",\"type\":\"address[]\"}],\"name\":\"setAppCandidateAddrs\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"admin\",\"type\":\"address\"}],\"name\":\"addAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"appId\",\"type\":\"string\"}],\"name\":\"getAppConsensusType\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"_isAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"appId\",\"type\":\"string\"}],\"name\":\"getAppCandidateAddrs\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"whoAdded\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"LogAddAdmin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"whoRemoved\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"admin\",\"type\":\"address\"}],\"name\":\"LogRemoveAdmin\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"}]"

// AppManagerBin is the compiled bytecode used for deploying new contracts.
const AppManagerBin = `0x608060405234801561001057600080fd5b5060038054600160a060020a031916331790819055604051600160a060020a0391909116906000907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0908290a3600354600160a060020a0316600090815260046020908152604091829020805460ff191660011790558151338082529181019190915281517fdf593c01aa2f7c955ab35aee0623fe0744c2117efb32343667f5b9660e9d5049929181900390910190a1610bb2806100cf6000396000f3006080604052600436106100a35763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416631785f53c81146100a85780632d830ce8146100dd5780632feae2d0146101525780633a34d6de146101b2578063668acc4a1461021657806370480275146102a8578063ae1d9067146102c9578063e9e43f1f14610338578063f2fde38b14610359578063fb068e151461037a575b600080fd5b3480156100b457600080fd5b506100c9600160a060020a0360043516610423565b604080519115158252519081900360200190f35b3480156100e957600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101369436949293602493928401919081908401838280828437509497506105139650505050505050565b60408051600160a060020a039092168252519081900360200190f35b34801561015e57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101b09436949293602493928401919081908401838280828437509497505050923560ff16935061058392505050565b005b3480156101be57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101b094369492936024939284019190819084018382808284375094975050509235600160a060020a0316935061061892505050565b34801561022257600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101b0943694929360249392840191908190840183828082843750506040805187358901803560208181028481018201909552818452989b9a9989019892975090820195509350839250850190849080828437509497506106c69650505050505050565b3480156102b457600080fd5b506100c9600160a060020a036004351661075d565b3480156102d557600080fd5b506040805160206004803580820135601f810184900484028501840190955284845261032294369492936024939284019190819084018382808284375094975061084b9650505050505050565b6040805160ff9092168252519081900360200190f35b34801561034457600080fd5b506100c9600160a060020a03600435166108b6565b34801561036557600080fd5b506101b0600160a060020a03600435166108cb565b34801561038657600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526103d39436949293602493928401919081908401838280828437509497506109399650505050505050565b60408051602080825283518183015283519192839290830191858101910280838360005b8381101561040f5781810151838201526020016103f7565b505050509050019250505060405180910390f35b600354600090600160a060020a03163314610488576040805160e560020a62461bcd02815260206004820152601760248201527f6572726f723a206f70206d7573742062792061646d696e000000000000000000604482015290519081900360640190fd5b600160a060020a03821660009081526004602052604090205460ff1615156001141561050b57600160a060020a038216600081815260046020908152604091829020805460ff1916905581513381529081019290925280517f472fe119cc78dd3474799f043c04071f7092335f7d6e1a30eb4998940828b3e89281900390910190a15b506001919050565b600080826040518082805190602001908083835b602083106105465780518252601f199092019160209182019101610527565b51815160209384036101000a6000190180199092169116179052920194855250604051938490030190922054600160a060020a0316949350505050565b3360009081526004602052604090205460ff1615156105a157600080fd5b806002836040518082805190602001908083835b602083106105d45780518252601f1990920191602091820191016105b5565b51815160209384036101000a60001901801990921691161790529201948552506040519384900301909220805460ff191660ff949094169390931790925550505050565b3360009081526004602052604090205460ff16151561063657600080fd5b806000836040518082805190602001908083835b602083106106695780518252601f19909201916020918201910161064a565b51815160209384036101000a60001901801990921691161790529201948552506040519384900301909220805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a03949094169390931790925550505050565b3360009081526004602052604090205460ff1615156106e457600080fd5b806001836040518082805190602001908083835b602083106107175780518252601f1990920191602091820191016106f8565b51815160209384036101000a600019018019909216911617905292019485525060405193849003810190932084516107589591949190910192509050610ae0565b505050565b600354600090600160a060020a031633146107c2576040805160e560020a62461bcd02815260206004820152601760248201527f6572726f723a206f70206d7573742062792061646d696e000000000000000000604482015290519081900360640190fd5b600160a060020a03821660009081526004602052604090205460ff16151561050b57600160a060020a038216600081815260046020908152604091829020805460ff1916600117905581513381529081019290925280517fdf593c01aa2f7c955ab35aee0623fe0744c2117efb32343667f5b9660e9d50499281900390910190a1506001919050565b60006002826040518082805190602001908083835b6020831061087f5780518252601f199092019160209182019101610860565b51815160209384036101000a600019018019909216911617905292019485525060405193849003019092205460ff16949350505050565b60046020526000908152604090205460ff1681565b600354600160a060020a0316331461092d576040805160e560020a62461bcd02815260206004820152601760248201527f6572726f723a206f70206d7573742062792061646d696e000000000000000000604482015290519081900360640190fd5b610936816109f1565b50565b60606001826040518082805190602001908083835b6020831061096d5780518252601f19909201916020918201910161094e565b51815160209384036101000a6000190180199092169116179052920194855250604080519485900382018520805480840287018401909252818652935091508301828280156109e557602002820191906000526020600020905b8154600160a060020a031681526001909101906020018083116109c7575b50505050509050919050565b600160a060020a0381161515610a77576040805160e560020a62461bcd02815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015290519081900360840190fd5b600354604051600160a060020a038084169216907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a36003805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b828054828255906000526020600020908101928215610b42579160200282015b82811115610b42578251825473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a03909116178255602090920191600190910190610b00565b50610b4e929150610b52565b5090565b610b8391905b80821115610b4e57805473ffffffffffffffffffffffffffffffffffffffff19168155600101610b58565b905600a165627a7a72305820da3b98de0337ef6e2b7564c3c5b04a5f120fccb4c912b0c20742650f27ca84b00029`

// DeployAppManager deploys a new Ethereum contract, binding an instance of AppManager to it.
func DeployAppManager(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *AppManager, error) {
	parsed, err := abi.JSON(strings.NewReader(AppManagerABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(AppManagerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AppManager{AppManagerCaller: AppManagerCaller{contract: contract}, AppManagerTransactor: AppManagerTransactor{contract: contract}, AppManagerFilterer: AppManagerFilterer{contract: contract}}, nil
}

// AppManager is an auto generated Go binding around an Ethereum contract.
type AppManager struct {
	AppManagerCaller     // Read-only binding to the contract
	AppManagerTransactor // Write-only binding to the contract
	AppManagerFilterer   // Log filterer for contract events
}

// AppManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type AppManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AppManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AppManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AppManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AppManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AppManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AppManagerSession struct {
	Contract     *AppManager       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AppManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AppManagerCallerSession struct {
	Contract *AppManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// AppManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AppManagerTransactorSession struct {
	Contract     *AppManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AppManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type AppManagerRaw struct {
	Contract *AppManager // Generic contract binding to access the raw methods on
}

// AppManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AppManagerCallerRaw struct {
	Contract *AppManagerCaller // Generic read-only contract binding to access the raw methods on
}

// AppManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AppManagerTransactorRaw struct {
	Contract *AppManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAppManager creates a new instance of AppManager, bound to a specific deployed contract.
func NewAppManager(address common.Address, backend bind.ContractBackend) (*AppManager, error) {
	contract, err := bindAppManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AppManager{AppManagerCaller: AppManagerCaller{contract: contract}, AppManagerTransactor: AppManagerTransactor{contract: contract}, AppManagerFilterer: AppManagerFilterer{contract: contract}}, nil
}

// NewAppManagerCaller creates a new read-only instance of AppManager, bound to a specific deployed contract.
func NewAppManagerCaller(address common.Address, caller bind.ContractCaller) (*AppManagerCaller, error) {
	contract, err := bindAppManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AppManagerCaller{contract: contract}, nil
}

// NewAppManagerTransactor creates a new write-only instance of AppManager, bound to a specific deployed contract.
func NewAppManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*AppManagerTransactor, error) {
	contract, err := bindAppManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AppManagerTransactor{contract: contract}, nil
}

// NewAppManagerFilterer creates a new log filterer instance of AppManager, bound to a specific deployed contract.
func NewAppManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*AppManagerFilterer, error) {
	contract, err := bindAppManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AppManagerFilterer{contract: contract}, nil
}

// bindAppManager binds a generic wrapper to an already deployed contract.
func bindAppManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AppManagerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AppManager *AppManagerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _AppManager.Contract.AppManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AppManager *AppManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AppManager.Contract.AppManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AppManager *AppManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AppManager.Contract.AppManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AppManager *AppManagerCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _AppManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AppManager *AppManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AppManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AppManager *AppManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AppManager.Contract.contract.Transact(opts, method, params...)
}

// IsAdmin is a free data retrieval call binding the contract method 0xe9e43f1f.
//
// Solidity: function _isAdmin( address) constant returns(bool)
func (_AppManager *AppManagerCaller) IsAdmin(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _AppManager.contract.Call(opts, out, "_isAdmin", arg0)
	return *ret0, err
}

// IsAdmin is a free data retrieval call binding the contract method 0xe9e43f1f.
//
// Solidity: function _isAdmin( address) constant returns(bool)
func (_AppManager *AppManagerSession) IsAdmin(arg0 common.Address) (bool, error) {
	return _AppManager.Contract.IsAdmin(&_AppManager.CallOpts, arg0)
}

// IsAdmin is a free data retrieval call binding the contract method 0xe9e43f1f.
//
// Solidity: function _isAdmin( address) constant returns(bool)
func (_AppManager *AppManagerCallerSession) IsAdmin(arg0 common.Address) (bool, error) {
	return _AppManager.Contract.IsAdmin(&_AppManager.CallOpts, arg0)
}

// GetAppCandidateAddrs is a free data retrieval call binding the contract method 0xfb068e15.
//
// Solidity: function getAppCandidateAddrs(appId string) constant returns(address[])
func (_AppManager *AppManagerCaller) GetAppCandidateAddrs(opts *bind.CallOpts, appId string) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _AppManager.contract.Call(opts, out, "getAppCandidateAddrs", appId)
	return *ret0, err
}

// GetAppCandidateAddrs is a free data retrieval call binding the contract method 0xfb068e15.
//
// Solidity: function getAppCandidateAddrs(appId string) constant returns(address[])
func (_AppManager *AppManagerSession) GetAppCandidateAddrs(appId string) ([]common.Address, error) {
	return _AppManager.Contract.GetAppCandidateAddrs(&_AppManager.CallOpts, appId)
}

// GetAppCandidateAddrs is a free data retrieval call binding the contract method 0xfb068e15.
//
// Solidity: function getAppCandidateAddrs(appId string) constant returns(address[])
func (_AppManager *AppManagerCallerSession) GetAppCandidateAddrs(appId string) ([]common.Address, error) {
	return _AppManager.Contract.GetAppCandidateAddrs(&_AppManager.CallOpts, appId)
}

// GetAppConsensusType is a free data retrieval call binding the contract method 0xae1d9067.
//
// Solidity: function getAppConsensusType(appId string) constant returns(uint8)
func (_AppManager *AppManagerCaller) GetAppConsensusType(opts *bind.CallOpts, appId string) (uint8, error) {
	var (
		ret0 = new(uint8)
	)
	out := ret0
	err := _AppManager.contract.Call(opts, out, "getAppConsensusType", appId)
	return *ret0, err
}

// GetAppConsensusType is a free data retrieval call binding the contract method 0xae1d9067.
//
// Solidity: function getAppConsensusType(appId string) constant returns(uint8)
func (_AppManager *AppManagerSession) GetAppConsensusType(appId string) (uint8, error) {
	return _AppManager.Contract.GetAppConsensusType(&_AppManager.CallOpts, appId)
}

// GetAppConsensusType is a free data retrieval call binding the contract method 0xae1d9067.
//
// Solidity: function getAppConsensusType(appId string) constant returns(uint8)
func (_AppManager *AppManagerCallerSession) GetAppConsensusType(appId string) (uint8, error) {
	return _AppManager.Contract.GetAppConsensusType(&_AppManager.CallOpts, appId)
}

// GetAppCreatorAddr is a free data retrieval call binding the contract method 0x2d830ce8.
//
// Solidity: function getAppCreatorAddr(appId string) constant returns(address)
func (_AppManager *AppManagerCaller) GetAppCreatorAddr(opts *bind.CallOpts, appId string) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _AppManager.contract.Call(opts, out, "getAppCreatorAddr", appId)
	return *ret0, err
}

// GetAppCreatorAddr is a free data retrieval call binding the contract method 0x2d830ce8.
//
// Solidity: function getAppCreatorAddr(appId string) constant returns(address)
func (_AppManager *AppManagerSession) GetAppCreatorAddr(appId string) (common.Address, error) {
	return _AppManager.Contract.GetAppCreatorAddr(&_AppManager.CallOpts, appId)
}

// GetAppCreatorAddr is a free data retrieval call binding the contract method 0x2d830ce8.
//
// Solidity: function getAppCreatorAddr(appId string) constant returns(address)
func (_AppManager *AppManagerCallerSession) GetAppCreatorAddr(appId string) (common.Address, error) {
	return _AppManager.Contract.GetAppCreatorAddr(&_AppManager.CallOpts, appId)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(admin address) returns(bool)
func (_AppManager *AppManagerTransactor) AddAdmin(opts *bind.TransactOpts, admin common.Address) (*types.Transaction, error) {
	return _AppManager.contract.Transact(opts, "addAdmin", admin)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(admin address) returns(bool)
func (_AppManager *AppManagerSession) AddAdmin(admin common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.AddAdmin(&_AppManager.TransactOpts, admin)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(admin address) returns(bool)
func (_AppManager *AppManagerTransactorSession) AddAdmin(admin common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.AddAdmin(&_AppManager.TransactOpts, admin)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(admin address) returns(bool)
func (_AppManager *AppManagerTransactor) RemoveAdmin(opts *bind.TransactOpts, admin common.Address) (*types.Transaction, error) {
	return _AppManager.contract.Transact(opts, "removeAdmin", admin)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(admin address) returns(bool)
func (_AppManager *AppManagerSession) RemoveAdmin(admin common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.RemoveAdmin(&_AppManager.TransactOpts, admin)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(admin address) returns(bool)
func (_AppManager *AppManagerTransactorSession) RemoveAdmin(admin common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.RemoveAdmin(&_AppManager.TransactOpts, admin)
}

// SetAppCandidateAddrs is a paid mutator transaction binding the contract method 0x668acc4a.
//
// Solidity: function setAppCandidateAddrs(appId string, addrs address[]) returns()
func (_AppManager *AppManagerTransactor) SetAppCandidateAddrs(opts *bind.TransactOpts, appId string, addrs []common.Address) (*types.Transaction, error) {
	return _AppManager.contract.Transact(opts, "setAppCandidateAddrs", appId, addrs)
}

// SetAppCandidateAddrs is a paid mutator transaction binding the contract method 0x668acc4a.
//
// Solidity: function setAppCandidateAddrs(appId string, addrs address[]) returns()
func (_AppManager *AppManagerSession) SetAppCandidateAddrs(appId string, addrs []common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.SetAppCandidateAddrs(&_AppManager.TransactOpts, appId, addrs)
}

// SetAppCandidateAddrs is a paid mutator transaction binding the contract method 0x668acc4a.
//
// Solidity: function setAppCandidateAddrs(appId string, addrs address[]) returns()
func (_AppManager *AppManagerTransactorSession) SetAppCandidateAddrs(appId string, addrs []common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.SetAppCandidateAddrs(&_AppManager.TransactOpts, appId, addrs)
}

// SetAppConsensusType is a paid mutator transaction binding the contract method 0x2feae2d0.
//
// Solidity: function setAppConsensusType(appId string, consensusType uint8) returns()
func (_AppManager *AppManagerTransactor) SetAppConsensusType(opts *bind.TransactOpts, appId string, consensusType uint8) (*types.Transaction, error) {
	return _AppManager.contract.Transact(opts, "setAppConsensusType", appId, consensusType)
}

// SetAppConsensusType is a paid mutator transaction binding the contract method 0x2feae2d0.
//
// Solidity: function setAppConsensusType(appId string, consensusType uint8) returns()
func (_AppManager *AppManagerSession) SetAppConsensusType(appId string, consensusType uint8) (*types.Transaction, error) {
	return _AppManager.Contract.SetAppConsensusType(&_AppManager.TransactOpts, appId, consensusType)
}

// SetAppConsensusType is a paid mutator transaction binding the contract method 0x2feae2d0.
//
// Solidity: function setAppConsensusType(appId string, consensusType uint8) returns()
func (_AppManager *AppManagerTransactorSession) SetAppConsensusType(appId string, consensusType uint8) (*types.Transaction, error) {
	return _AppManager.Contract.SetAppConsensusType(&_AppManager.TransactOpts, appId, consensusType)
}

// SetAppCreatorAddr is a paid mutator transaction binding the contract method 0x3a34d6de.
//
// Solidity: function setAppCreatorAddr(appId string, addr address) returns()
func (_AppManager *AppManagerTransactor) SetAppCreatorAddr(opts *bind.TransactOpts, appId string, addr common.Address) (*types.Transaction, error) {
	return _AppManager.contract.Transact(opts, "setAppCreatorAddr", appId, addr)
}

// SetAppCreatorAddr is a paid mutator transaction binding the contract method 0x3a34d6de.
//
// Solidity: function setAppCreatorAddr(appId string, addr address) returns()
func (_AppManager *AppManagerSession) SetAppCreatorAddr(appId string, addr common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.SetAppCreatorAddr(&_AppManager.TransactOpts, appId, addr)
}

// SetAppCreatorAddr is a paid mutator transaction binding the contract method 0x3a34d6de.
//
// Solidity: function setAppCreatorAddr(appId string, addr address) returns()
func (_AppManager *AppManagerTransactorSession) SetAppCreatorAddr(appId string, addr common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.SetAppCreatorAddr(&_AppManager.TransactOpts, appId, addr)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(newOwner address) returns()
func (_AppManager *AppManagerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _AppManager.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(newOwner address) returns()
func (_AppManager *AppManagerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.TransferOwnership(&_AppManager.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(newOwner address) returns()
func (_AppManager *AppManagerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _AppManager.Contract.TransferOwnership(&_AppManager.TransactOpts, newOwner)
}

// AppManagerLogAddAdminIterator is returned from FilterLogAddAdmin and is used to iterate over the raw logs and unpacked data for LogAddAdmin events raised by the AppManager contract.
type AppManagerLogAddAdminIterator struct {
	Event *AppManagerLogAddAdmin // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AppManagerLogAddAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AppManagerLogAddAdmin)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AppManagerLogAddAdmin)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AppManagerLogAddAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AppManagerLogAddAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AppManagerLogAddAdmin represents a LogAddAdmin event raised by the AppManager contract.
type AppManagerLogAddAdmin struct {
	WhoAdded common.Address
	NewAdmin common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterLogAddAdmin is a free log retrieval operation binding the contract event 0xdf593c01aa2f7c955ab35aee0623fe0744c2117efb32343667f5b9660e9d5049.
//
// Solidity: e LogAddAdmin(whoAdded address, newAdmin address)
func (_AppManager *AppManagerFilterer) FilterLogAddAdmin(opts *bind.FilterOpts) (*AppManagerLogAddAdminIterator, error) {

	logs, sub, err := _AppManager.contract.FilterLogs(opts, "LogAddAdmin")
	if err != nil {
		return nil, err
	}
	return &AppManagerLogAddAdminIterator{contract: _AppManager.contract, event: "LogAddAdmin", logs: logs, sub: sub}, nil
}

// WatchLogAddAdmin is a free log subscription operation binding the contract event 0xdf593c01aa2f7c955ab35aee0623fe0744c2117efb32343667f5b9660e9d5049.
//
// Solidity: e LogAddAdmin(whoAdded address, newAdmin address)
func (_AppManager *AppManagerFilterer) WatchLogAddAdmin(opts *bind.WatchOpts, sink chan<- *AppManagerLogAddAdmin) (event.Subscription, error) {

	logs, sub, err := _AppManager.contract.WatchLogs(opts, "LogAddAdmin")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AppManagerLogAddAdmin)
				if err := _AppManager.contract.UnpackLog(event, "LogAddAdmin", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// AppManagerLogRemoveAdminIterator is returned from FilterLogRemoveAdmin and is used to iterate over the raw logs and unpacked data for LogRemoveAdmin events raised by the AppManager contract.
type AppManagerLogRemoveAdminIterator struct {
	Event *AppManagerLogRemoveAdmin // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AppManagerLogRemoveAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AppManagerLogRemoveAdmin)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AppManagerLogRemoveAdmin)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AppManagerLogRemoveAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AppManagerLogRemoveAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AppManagerLogRemoveAdmin represents a LogRemoveAdmin event raised by the AppManager contract.
type AppManagerLogRemoveAdmin struct {
	WhoRemoved common.Address
	Admin      common.Address
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterLogRemoveAdmin is a free log retrieval operation binding the contract event 0x472fe119cc78dd3474799f043c04071f7092335f7d6e1a30eb4998940828b3e8.
//
// Solidity: e LogRemoveAdmin(whoRemoved address, admin address)
func (_AppManager *AppManagerFilterer) FilterLogRemoveAdmin(opts *bind.FilterOpts) (*AppManagerLogRemoveAdminIterator, error) {

	logs, sub, err := _AppManager.contract.FilterLogs(opts, "LogRemoveAdmin")
	if err != nil {
		return nil, err
	}
	return &AppManagerLogRemoveAdminIterator{contract: _AppManager.contract, event: "LogRemoveAdmin", logs: logs, sub: sub}, nil
}

// WatchLogRemoveAdmin is a free log subscription operation binding the contract event 0x472fe119cc78dd3474799f043c04071f7092335f7d6e1a30eb4998940828b3e8.
//
// Solidity: e LogRemoveAdmin(whoRemoved address, admin address)
func (_AppManager *AppManagerFilterer) WatchLogRemoveAdmin(opts *bind.WatchOpts, sink chan<- *AppManagerLogRemoveAdmin) (event.Subscription, error) {

	logs, sub, err := _AppManager.contract.WatchLogs(opts, "LogRemoveAdmin")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AppManagerLogRemoveAdmin)
				if err := _AppManager.contract.UnpackLog(event, "LogRemoveAdmin", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// AppManagerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the AppManager contract.
type AppManagerOwnershipTransferredIterator struct {
	Event *AppManagerOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AppManagerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AppManagerOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AppManagerOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AppManagerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AppManagerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AppManagerOwnershipTransferred represents a OwnershipTransferred event raised by the AppManager contract.
type AppManagerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: e OwnershipTransferred(previousOwner indexed address, newOwner indexed address)
func (_AppManager *AppManagerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*AppManagerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _AppManager.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &AppManagerOwnershipTransferredIterator{contract: _AppManager.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: e OwnershipTransferred(previousOwner indexed address, newOwner indexed address)
func (_AppManager *AppManagerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *AppManagerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _AppManager.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AppManagerOwnershipTransferred)
				if err := _AppManager.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
