// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"context"
	"github.com/CarLiveChainCo/goiov/internal/ethapi"
	"github.com/CarLiveChainCo/goiov/node"
	"math/big"

	"github.com/CarLiveChainCo/goiov/accounts"
	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/common/math"
	"github.com/CarLiveChainCo/goiov/core"
	"github.com/CarLiveChainCo/goiov/core/bloombits"
	"github.com/CarLiveChainCo/goiov/core/rawdb"
	"github.com/CarLiveChainCo/goiov/core/state"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/core/vm"
	"github.com/CarLiveChainCo/goiov/eth/downloader"
	"github.com/CarLiveChainCo/goiov/eth/gasprice"
	"github.com/CarLiveChainCo/goiov/ethdb"
	"github.com/CarLiveChainCo/goiov/event"
	"github.com/CarLiveChainCo/goiov/params"
	"github.com/CarLiveChainCo/goiov/rpc"
	"errors"
)

var ErrNoSideChain = errors.New("the side chain is not created. Please create the side chain by command 'eth.NewSideChain(appId)'")
var ErrAppIdExists = errors.New("this appId already exists")
var ErrMissingAppId = errors.New("the appid is missing")
var ErrNoSideMiner = errors.New("the side miner is not created. Please create the side chain by command 'eth.NewSideChain(appId)'")
var ErrWrongNumber = errors.New("wrong BlockNumber")
// EthAPIBackend implements ethapi.Backend for full nodes
type EthAPIBackend struct {
	eth *Ethereum
	gpo *gasprice.Oracle
}

func (b *EthAPIBackend) ChainConfig() *params.ChainConfig {
	return b.eth.chainConfig
}

func (b *EthAPIBackend) NodeConfig() *node.Config {
	return b.eth.nodeConfig
}

func (b *EthAPIBackend) BlockChain() ethapi.BlockChain {
	return b.eth.blockchain
}

func (b *EthAPIBackend) CurrentBlock() *types.Block {
	return b.eth.blockchain.CurrentBlock()
}

func (b *EthAPIBackend) SetHead(number uint64) {
	b.eth.protocolManager.downloader.Cancel()
	b.eth.blockchain.SetHead(number)
}

func (b *EthAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.eth.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock().Header(), nil
	}
	return b.eth.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *EthAPIBackend) SideHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber, appId string) (*types.Header, error) {
	if appId == "" {
		return b.HeaderByNumber(ctx, blockNr)
	}
	sideChain, ok := b.eth.sideChains[appId]
	if !ok || sideChain == nil {
		return nil, ErrNoSideChain
	}
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		if miner, ok := b.eth.sideMiner[appId]; miner != nil && ok {
			block := miner.PendingBlock()
			return block.Header(), nil
		}
		return nil, ErrNoSideMiner
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return sideChain.CurrentBlock().Header(), nil
	}
	return sideChain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *EthAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.eth.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock(), nil
	}
	return b.eth.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *EthAPIBackend) SideBlockByNumber(ctx context.Context, blockNr rpc.BlockNumber, appId string) (*types.Block, error) {
	if appId == "" {
		return b.BlockByNumber(ctx, blockNr)
	}
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		if sideMiner, ok := b.eth.sideMiner[appId]; sideMiner != nil && ok {
			return sideMiner.PendingBlock(), nil
		}
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		if sideChain, ok := b.eth.sideChains[appId]; sideChain != nil && ok {
			return sideChain.CurrentBlock(), nil
		}
	}
	if sideChain, ok := b.eth.sideChains[appId]; sideChain != nil && ok {
		return sideChain.GetBlockByNumber(uint64(blockNr)), nil
	} else {
		return nil, ErrNoSideChain
	}
}

func (b *EthAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.eth.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.eth.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *EthAPIBackend) SideStateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber, appId string) (*state.StateDB, *types.Header, error) {
	if appId == "" {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		sideMiner, ok := b.eth.sideMiner[appId]
		if sideMiner == nil || !ok {
			return nil, nil, ErrNoSideChain
		}
		block, state := sideMiner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.SideHeaderByNumber(ctx, blockNr, appId)
	if header == nil || err != nil {
		return nil, nil, err
	}
	sideChain, ok := b.eth.sideChains[appId]
	if sideChain == nil || !ok {
		return nil, nil, ErrNoSideChain
	}
	stateDb, err := sideChain.StateAt(header.Root)
	return stateDb, header, err
}

func (b *EthAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(hash), nil
}

func (b *EthAPIBackend) GetSideBlock(ctx context.Context, hash common.Hash, appId string) (*types.Block, error) {
	if appId == "" {
		return b.GetBlock(ctx, hash)
	}
	if sideChain, ok := b.eth.sideChains[appId]; sideChain != nil && ok {
		return sideChain.GetBlockByHash(hash), nil
	}
	return nil, ErrNoSideChain
}

func (b *EthAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.eth.chainDb, hash); number != nil {
		return rawdb.ReadReceipts(b.eth.chainDb, hash, *number), nil
	}
	return nil, nil
}

func (b *EthAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	number := rawdb.ReadHeaderNumber(b.eth.chainDb, hash)
	if number == nil {
		return nil, nil
	}
	receipts := rawdb.ReadReceipts(b.eth.chainDb, hash, *number)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *EthAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.eth.blockchain.GetTdByHash(blockHash)
}

func (b *EthAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	chain := b.eth.BlockChain()
	if header.Appid != "" {
		if sideChain, ok := b.eth.GetSideChains()[header.Appid]; !ok {
			return nil, nil, ErrNoSideChain
		}else{
			chain = sideChain
		}
	}
	context := core.NewEVMContext(msg, header, chain, nil)
	return vm.NewEVM(context, state, chain.Config(), vmCfg), vmError, nil
}

func (b *EthAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EthAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EthAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EthAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	if isMain, err := b.isMainTx(signedTx); err != nil {
		return err
	} else if isMain {
		return b.eth.txPool.AddLocal(signedTx)
	}
	txPool := b.eth.SideTxPool(signedTx.AppId())
	if txPool == nil {
		return ErrNoSideChain
	}
	return txPool.AddLocal(signedTx)
}
func (b *EthAPIBackend) isMainTx(signedTx *types.Transaction) (bool, error) {
	if signedTx.AppId() == "" {
		return true, nil
	}
	// 创建合约有：1：扩展合约		2：创建侧链合约	3：创建主链合约不建侧链(appId==nil)
	if signedTx.To() == nil {
		// 扩展合约，需要先引入侧链，并且作者必须相同
		stored := rawdb.ReadCanonicalHash(b.ChainDb(), 0, signedTx.AppId())
		if stored != (common.Hash{}) {
			if chain, ok := b.SideBlockChain(signedTx.AppId()); !ok {
				return false, core.ErrNoUseSideChain
			} else {
				signer := types.MakeSigner(chain.Config(), chain.CurrentBlock().Number())
				from, err := types.Sender(signer, signedTx)
				if err != nil {
					return false, err
				}
				config := rawdb.ReadChainConfig(b.eth.chainDb, stored, signedTx.AppId())
				if from != config.Author {
					return false, ErrAppIdExists
				}
			}
		} else {
			return true, nil
		}
	}
	return false, nil
}

func (b *EthAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.eth.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *EthAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.eth.txPool.Get(hash)
}

func (b *EthAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address, appid string) (uint64, error) {
	if appid != ""{
		txPool, ok := b.eth.sideTxPool[appid]
		if !ok {
			return 0, ErrNoSideChain
		}
		return txPool.State().GetNonce(addr), nil
	}
	return b.eth.txPool.State().GetNonce(addr), nil
}

func (b *EthAPIBackend) Stats() (pending int, queued int) {
	return b.eth.txPool.Stats()
}

func (b *EthAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.eth.TxPool().Content()
}

func (b *EthAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.eth.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *EthAPIBackend) Downloader() *downloader.Downloader {
	return b.eth.Downloader()
}

func (b *EthAPIBackend) ProtocolVersion() int {
	return b.eth.EthVersion()
}

func (b *EthAPIBackend) SuggestPrice(ctx context.Context, appId string) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx, appId)
}

func (b *EthAPIBackend) ChainDb() ethdb.Database {
	return b.eth.ChainDb()
}

func (b *EthAPIBackend) EventMux() *event.TypeMux {
	return b.eth.EventMux()
}

func (b *EthAPIBackend) AccountManager() *accounts.Manager {
	return b.eth.AccountManager()
}

func (b *EthAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.eth.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EthAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.eth.bloomRequests)
	}
}

func (b *EthAPIBackend) SideBlockChain(appId string) (*core.BlockChain, bool) {
	if appId == "" {
		return b.eth.blockchain, true
	}
	if block, ok := b.eth.sideChains[appId]; !ok {
		return nil, false
	} else {
		return block, true
	}
}

func (b *EthAPIBackend) SideTxPool(appId string) (*core.TxPool) {
	if appId == "" {
		return b.eth.txPool
	}
	if txPool, ok := b.eth.sideTxPool[appId]; !ok {
		return nil
	} else {
		return txPool
	}
}

func (b *EthAPIBackend) GetSideChains() map[string]*core.BlockChain {
	return b.eth.sideChains
}
