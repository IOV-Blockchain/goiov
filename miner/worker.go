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

package miner

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/consensus"
	"github.com/CarLiveChainCo/goiov/core"
	"github.com/CarLiveChainCo/goiov/core/state"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/core/vm"
	"github.com/CarLiveChainCo/goiov/ethdb"
	"github.com/CarLiveChainCo/goiov/event"
	"github.com/CarLiveChainCo/goiov/log"
	"github.com/CarLiveChainCo/goiov/params"
	"gopkg.in/fatih/set.v0"
	"github.com/CarLiveChainCo/goiov/accounts"
	"github.com/CarLiveChainCo/goiov/crypto"
	"github.com/CarLiveChainCo/goiov/core/rawdb"
	"reflect"
)

const (
	resultQueueSize  = 10
	miningLogAtDepth = 5

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
	// chainSideChanSize is the size of channel listening to ChainSideEvent.
	chainSideChanSize = 10
)

// Agent can register themself with the worker
type Agent interface {
	Work() chan<- *Work
	SetReturnCh(chan<- *Result)
	Stop()
	Start()
	GetHashRate() int64
}

// Work is the workers current environment and holds
// all of the current state information
type Work struct {
	config *params.ChainConfig
	signer types.Signer

	state     *state.StateDB // apply state changes here
	ancestors *set.Set       // ancestor set (used for checking uncle parent validity)
	family    *set.Set       // family set (used for checking uncle invalidity)
	uncles    *set.Set       // uncle set
	tcount    int            // tx count in cycle
	gasPool   *core.GasPool  // available gas used to pack transactions

	Block *types.Block // the new block

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt

	createdAt time.Time
}

type Result struct {
	Work  *Work
	Block *types.Block
}

// worker is the main object which takes care of applying messages to the new state
type worker struct {
	config *params.ChainConfig
	engine consensus.Engine

	mu sync.Mutex

	// update loop
	mux          *event.TypeMux
	txsCh        chan core.NewTxsEvent
	txsSub       event.Subscription
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription
	chainSideCh  chan core.ChainSideEvent
	chainSideSub event.Subscription
	wg           sync.WaitGroup

	agents map[Agent]struct{}
	recv   chan *Result

	eth     Backend
	chain   *core.BlockChain
	proc    core.Validator
	chainDb ethdb.Database

	coinbase common.Address
	extra    []byte

	currentMu sync.Mutex
	current   *Work

	snapshotMu    sync.RWMutex
	snapshotBlock *types.Block
	snapshotState *state.StateDB

	receiveBlockMu sync.Mutex

	uncleMu        sync.Mutex
	possibleUncles map[common.Hash]*types.Block

	unconfirmed *unconfirmedBlocks // set of locally mined blocks pending canonicalness confirmations

	// atomic status counters
	mining int32
	atWork int32

	appid string
}

func newWorker(config *params.ChainConfig, engine consensus.Engine, coinbase common.Address, eth Backend, mux *event.TypeMux, appId string) *worker {
	var chain *core.BlockChain
	var ok bool
	if chain, ok = eth.SideBlockChain(appId); !ok {
		return nil
	}

	worker := &worker{
		config:         config,
		engine:         engine,
		eth:            eth,
		mux:            mux,
		txsCh:          make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:    make(chan core.ChainHeadEvent, chainHeadChanSize),
		chainSideCh:    make(chan core.ChainSideEvent, chainSideChanSize),
		chainDb:        eth.ChainDb(),
		recv:           make(chan *Result, resultQueueSize),
		chain:          chain,
		proc:           chain.Validator(),
		possibleUncles: make(map[common.Hash]*types.Block),
		coinbase:       coinbase,
		agents:         make(map[Agent]struct{}),
		unconfirmed:    newUnconfirmedBlocks(chain, miningLogAtDepth),
		appid:          appId,
	}
	// Subscribe NewTxsEvent for tx pool
	if appId == "" {
		worker.txsSub = eth.TxPool().SubscribeNewTxsEvent(worker.txsCh)
	} else {
		worker.txsSub = eth.SideTxPool(appId).SubscribeNewTxsEvent(worker.txsCh)
	}
	// Subscribe events for blockchain
	worker.chainHeadSub = chain.SubscribeChainHeadEvent(worker.chainHeadCh)
	worker.chainSideSub = chain.SubscribeChainSideEvent(worker.chainSideCh)
	go worker.update()

	go worker.wait()
	worker.commitNewWork()

	return worker
}

func (self *worker) setEtherbase(addr common.Address) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.coinbase = addr
}

func (self *worker) setAppid(id string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.appid = id
}

func (self *worker) setExtra(extra []byte) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.extra = extra
}

func (self *worker) pending() (*types.Block, *state.StateDB) {
	if atomic.LoadInt32(&self.mining) == 0 {
		// return a snapshot to avoid contention on currentMu mutex
		self.snapshotMu.RLock()
		defer self.snapshotMu.RUnlock()
		return self.snapshotBlock, self.snapshotState.Copy()
	}

	self.currentMu.Lock()
	defer self.currentMu.Unlock()
	return self.current.Block, self.current.state.Copy()
}

func (self *worker) pendingBlock() *types.Block {
	if atomic.LoadInt32(&self.mining) == 0 {
		// return a snapshot to avoid contention on currentMu mutex
		self.snapshotMu.RLock()
		defer self.snapshotMu.RUnlock()
		return self.snapshotBlock
	}

	self.currentMu.Lock()
	defer self.currentMu.Unlock()
	return self.current.Block
}

func (self *worker) start() {
	self.mu.Lock()
	defer self.mu.Unlock()

	atomic.StoreInt32(&self.mining, 1)

	// spin up agents
	for agent := range self.agents {
		agent.Start()
	}
}

func (self *worker) stop() {
	self.wg.Wait()

	self.mu.Lock()
	defer self.mu.Unlock()
	if atomic.LoadInt32(&self.mining) == 1 {
		for agent := range self.agents {
			agent.Stop()
		}
	}
	atomic.StoreInt32(&self.mining, 0)
	atomic.StoreInt32(&self.atWork, 0)
}

func (self *worker) register(agent Agent) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.agents[agent] = struct{}{}
	agent.SetReturnCh(self.recv)
}

func (self *worker) unregister(agent Agent) {
	self.mu.Lock()
	defer self.mu.Unlock()
	delete(self.agents, agent)
	agent.Stop()
}

func (self *worker) update() {
	defer self.txsSub.Unsubscribe()
	defer self.chainHeadSub.Unsubscribe()
	defer self.chainSideSub.Unsubscribe()

	// set the delay equal to period if use alien consensus
	alienDelay := time.Duration(300) * time.Second
	if self.config.Alien != nil && self.config.Alien.Period > 0 {
		alienDelay = time.Duration(self.config.Alien.Period) * time.Second
	}
	for {
		// A real event arrived, process interesting content
		select {
		// Handle ChainHeadEvent
		case <-self.chainHeadCh:
			self.commitNewWork()

			// Handle ChainSideEvent
		case ev := <-self.chainSideCh:
			self.uncleMu.Lock()
			self.possibleUncles[ev.Block.Hash()] = ev.Block
			self.uncleMu.Unlock()

			// Handle NewTxsEvent
		case ev := <-self.txsCh:
			// Apply transactions to the pending state if we're not mining.
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current mining block. These transactions will
			// be automatically eliminated.
			if atomic.LoadInt32(&self.mining) == 0 {
				self.currentMu.Lock()
				txs := make(map[common.Address]types.Transactions)
				for _, tx := range ev.Txs {
					acc, err := types.Sender(self.current.signer, tx)
					if err != nil {
						continue
					}
					txs[acc] = append(txs[acc], tx)
				}
				txset := types.NewTransactionsByPriceAndNonce(self.current.signer, txs)
				self.current.commitTransactions(self.mux, txset, self.chain, self.coinbase)
				self.updateSnapshot()
				self.currentMu.Unlock()
			} else {
				// If we're mining, but nothing is being processed, wake on new transactions
				if self.config.Clique != nil && self.config.Clique.Period == 0 {
					self.commitNewWork()
				}
			}
		case <-time.After(alienDelay):
			// try to seal block in each period, even no new block received in dpos
			if self.config.Alien != nil && self.config.Alien.Period > 0 {
				self.commitNewWork()
			}

			// System stopped
		case <-self.txsSub.Err():
			return
		case <-self.chainHeadSub.Err():
			return
		case <-self.chainSideSub.Err():
			return
		}
	}
}

func (self *worker) receiveBlock(mustCommitNewWork bool, result *Result) bool {
	self.receiveBlockMu.Lock()
	defer self.receiveBlockMu.Unlock()

	atomic.AddInt32(&self.atWork, -1)

	if result == nil {
		return mustCommitNewWork
	}
	block := result.Block
	work := result.Work

	// Update the block hash in all logs since it is now available and not when the
	// receipt/log of individual transactions were created.
	for _, r := range work.receipts {
		for _, l := range r.Logs {
			l.BlockHash = block.Hash()
		}
	}
	for _, log := range work.state.Logs() {
		log.BlockHash = block.Hash()
	}
	stat, err := self.chain.WriteBlockWithState(block, work.receipts, work.state)
	if err != nil {
		log.Error("Failed writing block to chain", "err", err)
		return mustCommitNewWork
	}
	// check if canon block and write transactions
	if stat == core.CanonStatTy {
		// implicit by posting ChainHeadEvent
		mustCommitNewWork = false
	}
	// Broadcast the block and announce chain insertion event
	self.mux.Post(core.NewMinedBlockEvent{Block: block})
	var (
		events []interface{}
		logs   = work.state.Logs()
	)
	events = append(events, core.ChainEvent{Block: block, Hash: block.Hash(), Logs: logs})
	if stat == core.CanonStatTy {
		events = append(events, core.ChainHeadEvent{Block: block})
	}
	self.chain.PostChainEvents(events, logs)

	// Insert the block into the set of pending ones to wait for confirmations
	self.unconfirmed.Insert(block.NumberU64(), block.Hash())

	if mustCommitNewWork {
		self.commitNewWork()
	}

	return mustCommitNewWork
}

func (self *worker) wait() {
	for {
		mustCommitNewWork := true
		for result := range self.recv {
			mustCommitNewWork = self.receiveBlock(mustCommitNewWork, result)
		}
	}
}

// push sends a new work task to currently live miner agents.
func (self *worker) push(work *Work) {
	if atomic.LoadInt32(&self.mining) != 1 {
		return
	}
	for agent := range self.agents {
		atomic.AddInt32(&self.atWork, 1)
		if ch := agent.Work(); ch != nil {
			ch <- work
		}
	}
}

// makeCurrent creates a new environment for the current cycle.
func (self *worker) makeCurrent(parent *types.Block, header *types.Header) error {
	state, err := self.chain.StateAt(parent.Root())
	if err != nil {
		return err
	}
	work := &Work{
		config:    self.config,
		signer:    types.NewEIP155Signer(self.config.ChainId),
		state:     state,
		ancestors: set.New(),
		family:    set.New(),
		uncles:    set.New(),
		header:    header,
		createdAt: time.Now(),
	}

	// when 08 is processed ancestors contain 07 (quick block)
	for _, ancestor := range self.chain.GetBlocksFromHash(parent.Hash(), 7) {
		for _, uncle := range ancestor.Uncles() {
			work.family.Add(uncle.Hash())
		}
		work.family.Add(ancestor.Hash())
		work.ancestors.Add(ancestor.Hash())
	}

	// Keep track of transactions which return errors so they can be removed
	work.tcount = 0
	self.current = work
	return nil
}

func (self *worker) commitNewWork() {
	self.receiveBlockMu.Lock()
	defer self.receiveBlockMu.Unlock()
	self.mu.Lock()
	defer self.mu.Unlock()
	self.uncleMu.Lock()
	defer self.uncleMu.Unlock()
	self.currentMu.Lock()
	defer self.currentMu.Unlock()
	tstart := time.Now()
	parent := self.chain.CurrentBlock()

	tstamp := tstart.Unix()
	if parent.Time().Cmp(new(big.Int).SetInt64(tstamp)) >= 0 {
		tstamp = parent.Time().Int64() + 1
	}
	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); tstamp > now+1 {
		wait := time.Duration(tstamp-now) * time.Second
		log.Info("Mining too far in the future", "wait", common.PrettyDuration(wait))
		time.Sleep(wait)
	}

	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		GasLimit:   core.CalcGasLimit(parent),
		Extra:      self.extra,
		Time:       big.NewInt(tstamp),
		Appid:      self.appid,
	}
	// Only set the coinbase if we are mining (avoid spurious block rewards)
	if atomic.LoadInt32(&self.mining) == 1 {
		header.Coinbase = self.coinbase
	}
	if err := self.engine.Prepare(self.chain, header); err != nil {
		log.Error("Failed to prepare header for mining", "err", err)
		return
	}

	// Could potentially happen if starting to mine in an odd state.
	err := self.makeCurrent(parent, header)
	if err != nil {
		log.Error("Failed to create mining context", "err", err)
		return
	}
	// Create the current work task and check any fork transitions needed
	work := self.current
	//pending, err := self.eth.TxPool().Pending()
	txPool := self.eth.SideTxPool(self.appid)
	if txPool == nil {
		return
	}
	pending, err := txPool.Pending()
	if err != nil {
		log.Error("Failed to fetch pending transactions", "err", err)
		return
	}

	//部署侧链的主合约
	var contranctTx *types.Transaction
	if self.appid != "" && header.Number.Cmp(big.NewInt(1)) == 0 && self.coinbase == self.chain.Config().Author {
		if contranctTx, err = self.doSideContract(); err != nil {
			return
		}
		pending = make(map[common.Address]types.Transactions)
		pending[self.coinbase] = append(pending[self.coinbase], contranctTx)
	}

	txs := types.NewTransactionsByPriceAndNonce(self.current.signer, pending)
	work.commitTransactions(self.mux, txs, self.chain, self.coinbase)

	// compute uncles for the new block.
	var (
		uncles    []*types.Header
		badUncles []common.Hash
	)
	for hash, uncle := range self.possibleUncles {
		if len(uncles) == 2 {
			break
		}
		if err := self.commitUncle(work, uncle.Header()); err != nil {
			log.Trace("Bad uncle found and will be removed", "hash", hash)
			log.Trace(fmt.Sprint(uncle))

			badUncles = append(badUncles, hash)
		} else {
			log.Debug("Committing new uncle to block", "hash", hash)
			uncles = append(uncles, uncle.Header())
		}
	}
	for _, hash := range badUncles {
		delete(self.possibleUncles, hash)
	}
	// Create the new block to seal with the consensus engine
	if work.Block, err = self.engine.Finalize(self.chain, header, work.state, work.txs, uncles, work.receipts); err != nil {
		log.Error("Failed to finalize block for sealing", "err", err)
		return
	}
	// We only care about logging if we're actually mining.
	if atomic.LoadInt32(&self.mining) == 1 {
		log.Info("Commit new mining work", "appId", self.appid, "number", work.Block.Number(), "txs", work.tcount, "uncles", len(uncles), "elapsed", common.PrettyDuration(time.Since(tstart)))
		self.unconfirmed.Shift(work.Block.NumberU64() - 1)
	}
	self.push(work)
	self.updateSnapshot()


	if self.config.Alien != nil && !reflect.ValueOf(self.eth.Miner()).IsNil() && self.eth.IsMining() && !self.config.Alien.SideChain {
		err = self.sendConfirmTx(parent.Number())
		if err != nil {
			log.Info("Fail to Sign the transaction by coinbase", "err", err)
		}
	}

}

func (self *worker) doSideContract() (*types.Transaction, error) {
	if self.appid == "" {
		return nil, nil
	}

	//获取侧链作者
	wallets := self.eth.AccountManager().Wallets()
	if len(wallets) == 0 {
		return nil, errors.New("No wallets")
	}
	var authorWallet accounts.Wallet
	var authorAccount accounts.Account
	for _, wallet := range wallets {
		if len(wallet.Accounts()) == 0 {
			continue
		}
		for _, account := range wallet.Accounts() {
			if account.Address == self.coinbase {
				if account.Address != self.chain.Config().Author {
					return nil, errors.New("blocks with number 1 must be dug out by the author")
				}
				authorWallet = wallet
				authorAccount = account
				break
			}
		}
	}

	//获取交易
	tx, _, _, _ := rawdb.ReadTransaction(self.chain.Getdb(),self.chain.Config().TxHash)
	tmpTx := types.NewContractCreation(1, big.NewInt(0), uint64(1000000), big.NewInt(10000), tx.Data(), self.appid)

	// 签名交易
	var chainId *big.Int
	chain, ok := self.eth.SideBlockChain(self.appid)
	if !ok {
		return nil, core.ErrNoSideChain
	}
	if chain.Config().IsEIP155(big.NewInt(0)) {
		chainId = self.config.ChainId
	}
	signedTx, err := authorWallet.SignTx(authorAccount, tmpTx, chainId)
	if err != nil {
		return nil, err
	}

	log.Info("sideContractTx", "addr", crypto.CreateAddress(self.chain.Config().Author, 1, self.appid), "hash", signedTx.Hash())
	return signedTx, nil
}

// For PBFT of alien consensus, the follow code may move later
// After confirm a new block, the miner send a custom transaction to self, which value is 0
// and data like "ufo:1:event:confirm:123" (ufo is prefix, 1 is version, 123 is block number
// to let the next signer now this signer already confirm this block
func (self *worker) sendConfirmTx(blockNumber *big.Int) error {
	wallets := self.eth.AccountManager().Wallets()
	// wallets check
	if len(wallets) == 0 {
		return errors.New("No wallets")
	}
	for _, wallet := range wallets {
		if len(wallet.Accounts()) == 0 {
			continue
		} else {
			for _, account := range wallet.Accounts() {

				if account.Address == self.coinbase {
					nonce := self.snapshotState.GetNonce(account.Address)
					tmpTx := types.NewTransaction(nonce, account.Address, big.NewInt(0), uint64(100000), big.NewInt(10000), []byte(fmt.Sprintf("ufo:1:event:confirm:%d", blockNumber)), self.appid)
					var chainId *big.Int
					if self.eth.BlockChain().Config().IsEIP155(blockNumber) {
						chainId = self.eth.BlockChain().Config().ChainId
					}
					txPool := self.eth.TxPool()
					if self.appid != "" {
						chain, _ := self.eth.SideBlockChain(self.appid)
						if chain.Config().IsEIP155(blockNumber) {
							chainId = self.config.ChainId
						}
						txPool = self.eth.SideTxPool(self.appid)
					}
					signedTx, err := wallet.SignTx(account, tmpTx, chainId)
					if err != nil {
						return err
					} else {
						err = txPool.AddLocal(signedTx)
						if err != nil {
							return err
						}
					}
					return nil
				}
			}
		}

	}
	return nil
}

func (self *worker) commitUncle(work *Work, uncle *types.Header) error {
	hash := uncle.Hash()
	if work.uncles.Has(hash) {
		return fmt.Errorf("uncle not unique")
	}
	if !work.ancestors.Has(uncle.ParentHash) {
		return fmt.Errorf("uncle's parent unknown (%x)", uncle.ParentHash[0:4])
	}
	if work.family.Has(hash) {
		return fmt.Errorf("uncle already in family (%x)", hash)
	}
	work.uncles.Add(uncle.Hash())
	return nil
}

func (self *worker) updateSnapshot() {
	self.snapshotMu.Lock()
	defer self.snapshotMu.Unlock()

	self.snapshotBlock = types.NewBlock(
		self.current.header,
		self.current.txs,
		nil,
		self.current.receipts,
	)
	self.snapshotState = self.current.state.Copy()
}

func (env *Work) commitTransactions(mux *event.TypeMux, txs *types.TransactionsByPriceAndNonce, bc *core.BlockChain, coinbase common.Address) {
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(env.header.GasLimit)
	}
	var coalescedLogs []*types.Log

	for {
		// If we don't have enough gas for any further transactions then we're done
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(env.signer, tx)
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !env.config.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", env.config.EIP155Block)
			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.Prepare(tx.Hash(), common.Hash{}, env.tcount)

		err, logs := env.commitTransaction(tx, bc, coinbase, env.gasPool)
		switch err {
		case core.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case core.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case core.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}

	if len(coalescedLogs) > 0 || env.tcount > 0 {
		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		go func(logs []*types.Log, tcount int) {
			if len(logs) > 0 {
				mux.Post(core.PendingLogsEvent{Logs: logs})
			}
			if tcount > 0 {
				mux.Post(core.PendingStateEvent{})
			}
		}(cpy, env.tcount)
	}
}

func (env *Work) commitTransaction(tx *types.Transaction, bc *core.BlockChain, coinbase common.Address, gp *core.GasPool) (error, []*types.Log) {
	snap := env.state.Snapshot()

	receipt, _, err := core.ApplyTransaction(env.config, bc, &coinbase, gp, env.state, env.header, tx, &env.header.GasUsed, vm.Config{})
	if err != nil {
		env.state.RevertToSnapshot(snap)
		return err, nil
	}
	env.txs = append(env.txs, tx)
	env.receipts = append(env.receipts, receipt)

	return nil, receipt.Logs
}
