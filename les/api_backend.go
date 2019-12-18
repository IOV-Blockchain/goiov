// Copyright 2016 The go-ethereum Authors
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

package les

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
	"github.com/CarLiveChainCo/goiov/light"
	"github.com/CarLiveChainCo/goiov/params"
	"github.com/CarLiveChainCo/goiov/rpc"
)

type LesApiBackend struct {
	eth *LightEthereum
	gpo *gasprice.Oracle
}

func (b *LesApiBackend) ChainConfig() *params.ChainConfig {
	return b.eth.chainConfig
}

func (b *LesApiBackend) NodeConfig() *node.Config {
	return b.eth.nodeConfig
}

func (b *LesApiBackend) BlockChain() ethapi.BlockChain {
	return b.eth.blockchain
}

func (b *LesApiBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(b.eth.BlockChain().CurrentHeader())
}

func (b *LesApiBackend) SetHead(number uint64) {
	b.eth.protocolManager.downloader.Cancel()
	b.eth.blockchain.SetHead(number)
}

func (b *LesApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		return b.eth.blockchain.CurrentHeader(), nil
	}

	return b.eth.blockchain.GetHeaderByNumberOdr(ctx, uint64(blockNr))
}

func (b *LesApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, err
	}
	return b.GetBlock(ctx, header.Hash())
}

func (b *LesApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	return light.NewState(ctx, header, b.eth.odr), header, nil
}

func (b *LesApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(ctx, blockHash)
}

func (b *LesApiBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.eth.chainDb, hash); number != nil {
		return light.GetBlockReceipts(ctx, b.eth.odr, hash, *number)
	}
	return nil, nil
}

func (b *LesApiBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	if number := rawdb.ReadHeaderNumber(b.eth.chainDb, hash); number != nil {
		return light.GetBlockLogs(ctx, b.eth.odr, hash, *number)
	}
	return nil, nil
}

func (b *LesApiBackend) GetTd(hash common.Hash) *big.Int {
	return b.eth.blockchain.GetTdByHash(hash)
}

func (b *LesApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	context := core.NewEVMContext(msg, header, b.eth.blockchain, nil)
	return vm.NewEVM(context, state, b.eth.chainConfig, vmCfg), state.Error, nil
}

func (b *LesApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.eth.txPool.Add(ctx, signedTx)
}

func (b *LesApiBackend) RemoveTx(txHash common.Hash) {
	b.eth.txPool.RemoveTx(txHash)
}

func (b *LesApiBackend) GetPoolTransactions() (types.Transactions, error) {
	return b.eth.txPool.GetTransactions()
}

func (b *LesApiBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	return b.eth.txPool.GetTransaction(txHash)
}

func (b *LesApiBackend) GetPoolNonce(ctx context.Context, addr common.Address, appid string) (uint64, error) {
	return b.eth.txPool.GetNonce(ctx, addr)
}

func (b *LesApiBackend) Stats() (pending int, queued int) {
	return b.eth.txPool.Stats(), 0
}

func (b *LesApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.eth.txPool.Content()
}

func (b *LesApiBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.eth.txPool.SubscribeNewTxsEvent(ch)
}

func (b *LesApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.eth.blockchain.SubscribeChainEvent(ch)
}

func (b *LesApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.eth.blockchain.SubscribeChainHeadEvent(ch)
}

func (b *LesApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.eth.blockchain.SubscribeChainSideEvent(ch)
}

func (b *LesApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.blockchain.SubscribeLogsEvent(ch)
}

func (b *LesApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.eth.blockchain.SubscribeRemovedLogsEvent(ch)
}

func (b *LesApiBackend) Downloader() *downloader.Downloader {
	return b.eth.Downloader()
}

func (b *LesApiBackend) ProtocolVersion() int {
	return b.eth.LesVersion() + 10000
}

func (b *LesApiBackend) SuggestPrice(ctx context.Context , appId string) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx , appId)
}

func (b *LesApiBackend) ChainDb() ethdb.Database {
	return b.eth.chainDb
}

func (b *LesApiBackend) EventMux() *event.TypeMux {
	return b.eth.eventMux
}

func (b *LesApiBackend) AccountManager() *accounts.Manager {
	return b.eth.accountManager
}

func (b *LesApiBackend) BloomStatus() (uint64, uint64) {
	if b.eth.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := b.eth.bloomIndexer.Sections()
	return light.BloomTrieFrequency, sections
}

func (b *LesApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.eth.bloomRequests)
	}
}

func (b *LesApiBackend) SideBlockChain(appId string) (*core.BlockChain, bool) {
	block, ok := b.eth.sideChains[appId]
	return block, ok
}

func (b *LesApiBackend) GetSideBlock(ctx context.Context, blockHash common.Hash, appId string) (*types.Block, error) {
	return nil, nil
}

func (b *LesApiBackend) SideStateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber, appId string) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}

func (b *LesApiBackend) SideBlockByNumber(ctx context.Context, blockNr rpc.BlockNumber, appid string) (*types.Block, error) {
	return nil, nil
}
func (b *LesApiBackend) SideTxPool(appId string) (*core.TxPool){
	return nil
}
func (b *LesApiBackend) GetSideChains() map[string]*core.BlockChain{
	return b.eth.sideChains
}

func (b *LesApiBackend) SideHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber, appId string) (*types.Header, error){
	return nil,nil
}