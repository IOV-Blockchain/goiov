package appmanager

import (
	"github.com/CarLiveChainCo/goiov/accounts/abi/bind"
	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/contracts/appmanager/contract"
)

var (
	// TODO 在部署应用管理合约之后替换正确的合约地址
	MainNetAppManagerAddress = common.HexToAddress("0xe1145ba6594ba07adf68a7337b06f1404b4a6863")
	TestNetAppManagerAddress = common.HexToAddress("0x0c74100427635286c992240f2ce8977c1a539069")
	RealAppManagerAddress    = MainNetAppManagerAddress
)

type AppManager struct {
	*contract.AppManagerSession
	contractBackend bind.ContractBackend
}

func NewAppManager(/*transactOpts *bind.TransactOpts,*/ contractBackend bind.ContractBackend) (*AppManager, error) {
	appManager, err := contract.NewAppManager(RealAppManagerAddress, contractBackend)
	if err != nil {
		return nil, err
	}

	return &AppManager{
		&contract.AppManagerSession{
			Contract:     appManager,
			//TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}
