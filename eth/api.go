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
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/common/hexutil"
	"github.com/CarLiveChainCo/goiov/consensus"
	"github.com/CarLiveChainCo/goiov/core"
	"github.com/CarLiveChainCo/goiov/core/rawdb"
	"github.com/CarLiveChainCo/goiov/core/state"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/core/vm"
	"github.com/CarLiveChainCo/goiov/eth/downloader"
	"github.com/CarLiveChainCo/goiov/ethdb"
	"github.com/CarLiveChainCo/goiov/log"
	"github.com/CarLiveChainCo/goiov/miner"
	"github.com/CarLiveChainCo/goiov/params"
	"github.com/CarLiveChainCo/goiov/rlp"
	"github.com/CarLiveChainCo/goiov/rpc"
	"github.com/CarLiveChainCo/goiov/trie"
	"io"
	"math/big"
	"os"
	"strings"
	"github.com/CarLiveChainCo/goiov/crypto"
)

// PublicEthereumAPI provides an API to access Ethereum full node-related
// information.
type PublicEthereumAPI struct {
	e *Ethereum
}

// NewPublicEthereumAPI creates a new Ethereum protocol API for full nodes.
func NewPublicEthereumAPI(e *Ethereum) *PublicEthereumAPI {
	return &PublicEthereumAPI{e}
}

// Etherbase is the address that mining rewards will be send to
func (api *PublicEthereumAPI) Etherbase() (common.Address, error) {
	return api.e.Etherbase()
}

// Coinbase is the address that mining rewards will be send to (alias for Etherbase)
func (api *PublicEthereumAPI) Coinbase() (common.Address, error) {
	return api.Etherbase()
}

// Hashrate returns the POW hashrate
func (api *PublicEthereumAPI) Hashrate() hexutil.Uint64 {
	return hexutil.Uint64(api.e.Miner().HashRate())
}

// PublicMinerAPI provides an API to control the miner.
// It offers only methods that operate on data that pose no security risk when it is publicly accessible.
type PublicMinerAPI struct {
	e     *Ethereum
	agent *miner.RemoteAgent
}

// NewPublicMinerAPI create a new PublicMinerAPI instance.
func NewPublicMinerAPI(e *Ethereum) *PublicMinerAPI {
	agent := miner.NewRemoteAgent(e.BlockChain(), e.Engine())
	e.Miner().Register(agent)

	return &PublicMinerAPI{e, agent}
}

// Mining returns an indication if this node is currently mining.
func (api *PublicMinerAPI) Mining() bool {
	return api.e.IsMining()
}

// SubmitWork can be used by external miner to submit their POW solution. It returns an indication if the work was
// accepted. Note, this is not an indication if the provided work was valid!
func (api *PublicMinerAPI) SubmitWork(nonce types.BlockNonce, solution, digest common.Hash) bool {
	return api.agent.SubmitWork(nonce, digest, solution)
}

// GetWork returns a work package for external miner. The work package consists of 3 strings
// result[0], 32 bytes hex encoded current block header pow-hash
// result[1], 32 bytes hex encoded seed hash used for DAG
// result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
func (api *PublicMinerAPI) GetWork() ([3]string, error) {
	if !api.e.IsMining() {
		if err := api.e.StartMining(false, ""); err != nil {
			return [3]string{}, err
		}
	}
	work, err := api.agent.GetWork()
	if err != nil {
		return work, fmt.Errorf("mining not ready: %v", err)
	}
	return work, nil
}

// SubmitHashrate can be used for remote miners to submit their hash rate. This enables the node to report the combined
// hash rate of all miners which submit work through this node. It accepts the miner hash rate and an identifier which
// must be unique between nodes.
func (api *PublicMinerAPI) SubmitHashrate(hashrate hexutil.Uint64, id common.Hash) bool {
	api.agent.SubmitHashrate(id, uint64(hashrate))
	return true
}

// PrivateMinerAPI provides private RPC methods to control the miner.
// These methods can be abused by external users and must be considered insecure for use by untrusted users.
type PrivateMinerAPI struct {
	e *Ethereum
}

// NewPrivateMinerAPI create a new RPC service which controls the miner of this node.
func NewPrivateMinerAPI(e *Ethereum) *PrivateMinerAPI {
	return &PrivateMinerAPI{e: e}
}

// Start the miner with the given number of threads. If threads is nil the number
// of workers started is equal to the number of logical CPUs that are usable by
// this process. If mining is already running, this method adjust the number of
// threads allowed to use.
func (api *PrivateMinerAPI) Start(threads *int, id string) error {
	// Set the number of threads if the seal engine supports it

	if threads == nil {
		threads = new(int)
	} else if *threads == 0 {
		*threads = -1 // Disable the miner from within
	}

	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := api.e.engine.(threaded); ok {
		log.Info("Updated mining threads", "threads", *threads)
		th.SetThreads(*threads)
	}
	// Start the miner and return
	if !api.e.IsMining() {
		// Propagate the initial price point to the transaction pool
		api.e.lock.RLock()
		price := api.e.gasPrice
		api.e.lock.RUnlock()

		api.e.txPool.SetGasPrice(price)
		return api.e.StartMining(true, id)
	}
	return nil
}

// Stop the miner
func (api *PrivateMinerAPI) Stop() bool {
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := api.e.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	api.e.StopMining()
	return true
}

func (api *PrivateMinerAPI) StopSide(appId string) bool {
	if _, ok := api.e.SideBlockChain(appId); !ok {
		log.Error("the side chain is not created. Please create the side chain by command 'eth.NewSideChain(appId)'")
		return false
	}
	type threaded interface {
		SetThreads(threads int)
	}
	if sideMiner, ok := api.e.sideMiner[appId]; sideMiner != nil && ok {
		if th, ok := sideMiner.GetEngine().(threaded); ok {
			th.SetThreads(-1)
		}
		sideMiner.Stop()
		//delete(api.e.sideTxPool, appId)
		return true
	}
	return false
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (api *PrivateMinerAPI) SetExtra(extra string) (bool, error) {
	if err := api.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (api *PrivateMinerAPI) SetGasPrice(gasPrice hexutil.Big) bool {
	api.e.lock.Lock()
	api.e.gasPrice = (*big.Int)(&gasPrice)
	api.e.lock.Unlock()

	api.e.txPool.SetGasPrice((*big.Int)(&gasPrice))
	return true
}

// SetEtherbase sets the etherbase of the miner
func (api *PrivateMinerAPI) SetEtherbase(etherbase common.Address) bool {
	log.Info("api.go SetEtherbase","etherbase: ",etherbase)

	wallets := api.e.AccountManager().Wallets()
	for i := 0; i < len(wallets); i++ {
		accounts := wallets[i].Accounts()

		for j := 0; j < len(accounts); j++ {
			if etherbase.Str() == accounts[j].Address.Str() {
				api.e.SetEtherbase(etherbase)
				return true
			}
		}
	}

	return false
}

// GetHashrate returns the current hashrate of the miner.
func (api *PrivateMinerAPI) GetHashrate() uint64 {
	return uint64(api.e.miner.HashRate())
}

// PrivateAdminAPI is the collection of Ethereum full node-related APIs
// exposed over the private admin endpoint.
type PrivateAdminAPI struct {
	eth *Ethereum
}

// NewPrivateAdminAPI creates a new API definition for the full node private
// admin methods of the Ethereum service.
func NewPrivateAdminAPI(eth *Ethereum) *PrivateAdminAPI {
	return &PrivateAdminAPI{eth: eth}
}

// ExportChain exports the current blockchain into a local file.
func (api *PrivateAdminAPI) ExportChain(file string) (bool, error) {
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer out.Close()

	var writer io.Writer = out
	if strings.HasSuffix(file, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	// Export the blockchain
	if err := api.eth.BlockChain().Export(writer); err != nil {
		return false, err
	}
	return true, nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash(), b.NumberU64()) {
			return false
		}
	}

	return true
}

// ImportChain imports a blockchain from a local file.
func (api *PrivateAdminAPI) ImportChain(file string) (bool, error) {
	// Make sure the can access the file to import
	in, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer in.Close()

	var reader io.Reader = in
	if strings.HasSuffix(file, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return false, err
		}
	}

	// Run actual the import in pre-configured batches
	stream := rlp.NewStream(reader, 0)

	blocks, index := make([]*types.Block, 0, 2500), 0
	for batch := 0; ; batch++ {
		// Load a batch of blocks from the input file
		for len(blocks) < cap(blocks) {
			block := new(types.Block)
			if err := stream.Decode(block); err == io.EOF {
				break
			} else if err != nil {
				return false, fmt.Errorf("block %d: failed to parse: %v", index, err)
			}
			blocks = append(blocks, block)
			index++
		}
		if len(blocks) == 0 {
			break
		}

		if hasAllBlocks(api.eth.BlockChain(), blocks) {
			blocks = blocks[:0]
			continue
		}
		// Import the batch and reset the buffer
		if _, err := api.eth.BlockChain().InsertChain(blocks); err != nil {
			return false, fmt.Errorf("batch %d: failed to insert: %v", batch, err)
		}
		blocks = blocks[:0]
	}
	return true, nil
}

// PublicDebugAPI is the collection of Ethereum full node APIs exposed
// over the public debugging endpoint.
type PublicDebugAPI struct {
	eth *Ethereum
}

// NewPublicDebugAPI creates a new API definition for the full node-
// related public debug methods of the Ethereum service.
func NewPublicDebugAPI(eth *Ethereum) *PublicDebugAPI {
	return &PublicDebugAPI{eth: eth}
}

// DumpBlock retrieves the entire state of the database at a given block.
func (api *PublicDebugAPI) DumpBlock(blockNr rpc.BlockNumber) (state.Dump, error) {
	if blockNr == rpc.PendingBlockNumber {
		// If we're dumping the pending state, we need to request
		// both the pending block as well as the pending state from
		// the miner and operate on those
		_, stateDb := api.eth.miner.Pending()
		return stateDb.RawDump(), nil
	}
	var block *types.Block
	if blockNr == rpc.LatestBlockNumber {
		block = api.eth.blockchain.CurrentBlock()
	} else {
		block = api.eth.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return state.Dump{}, fmt.Errorf("block #%d not found", blockNr)
	}
	stateDb, err := api.eth.BlockChain().StateAt(block.Root())
	if err != nil {
		return state.Dump{}, err
	}
	return stateDb.RawDump(), nil
}

// PrivateDebugAPI is the collection of Ethereum full node APIs exposed over
// the private debugging endpoint.
type PrivateDebugAPI struct {
	config *params.ChainConfig
	eth    *Ethereum
}

// NewPrivateDebugAPI creates a new API definition for the full node-related
// private debug methods of the Ethereum service.
func NewPrivateDebugAPI(config *params.ChainConfig, eth *Ethereum) *PrivateDebugAPI {
	return &PrivateDebugAPI{config: config, eth: eth}
}

// Preimage is a debug API function that returns the preimage for a sha3 hash, if known.
func (api *PrivateDebugAPI) Preimage(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	if preimage := rawdb.ReadPreimage(api.eth.ChainDb(), hash); preimage != nil {
		return preimage, nil
	}
	return nil, errors.New("unknown preimage")
}

// GetBadBLocks returns a list of the last 'bad blocks' that the client has seen on the network
// and returns them as a JSON list of block-hashes
func (api *PrivateDebugAPI) GetBadBlocks(ctx context.Context) ([]core.BadBlockArgs, error) {
	return api.eth.BlockChain().BadBlocks()
}

// StorageRangeResult is the result of a debug_storageRangeAt API call.
type StorageRangeResult struct {
	Storage storageMap   `json:"storage"`
	NextKey *common.Hash `json:"nextKey"` // nil if Storage includes the last key in the trie.
}

type storageMap map[common.Hash]storageEntry

type storageEntry struct {
	Key   *common.Hash `json:"key"`
	Value common.Hash  `json:"value"`
}

// StorageRangeAt returns the storage at the given block height and transaction index.
func (api *PrivateDebugAPI) StorageRangeAt(ctx context.Context, blockHash common.Hash, txIndex int, contractAddress common.Address, keyStart hexutil.Bytes, maxResult int) (StorageRangeResult, error) {
	_, _, statedb, err := api.computeTxEnv(blockHash, txIndex, 0)
	if err != nil {
		return StorageRangeResult{}, err
	}
	st := statedb.StorageTrie(contractAddress)
	if st == nil {
		return StorageRangeResult{}, fmt.Errorf("account %x doesn't exist", contractAddress)
	}
	return storageRangeAt(st, keyStart, maxResult)
}

func storageRangeAt(st state.Trie, start []byte, maxResult int) (StorageRangeResult, error) {
	it := trie.NewIterator(st.NodeIterator(start))
	result := StorageRangeResult{Storage: storageMap{}}
	for i := 0; i < maxResult && it.Next(); i++ {
		_, content, _, err := rlp.Split(it.Value)
		if err != nil {
			return StorageRangeResult{}, err
		}
		e := storageEntry{Value: common.BytesToHash(content)}
		if preimage := st.GetKey(it.Key); preimage != nil {
			preimage := common.BytesToHash(preimage)
			e.Key = &preimage
		}
		result.Storage[common.BytesToHash(it.Key)] = e
	}
	// Add the 'next key' so clients can continue downloading.
	if it.Next() {
		next := common.BytesToHash(it.Key)
		result.NextKey = &next
	}
	return result, nil
}

// GetModifiedAccountsByumber returns all accounts that have changed between the
// two blocks specified. A change is defined as a difference in nonce, balance,
// code hash, or storage hash.
//
// With one parameter, returns the list of accounts modified in the specified block.
func (api *PrivateDebugAPI) GetModifiedAccountsByNumber(startNum uint64, endNum *uint64) ([]common.Address, error) {
	var startBlock, endBlock *types.Block

	startBlock = api.eth.blockchain.GetBlockByNumber(startNum)
	if startBlock == nil {
		return nil, fmt.Errorf("start block %x not found", startNum)
	}

	if endNum == nil {
		endBlock = startBlock
		startBlock = api.eth.blockchain.GetBlockByHash(startBlock.ParentHash())
		if startBlock == nil {
			return nil, fmt.Errorf("block %x has no parent", endBlock.Number())
		}
	} else {
		endBlock = api.eth.blockchain.GetBlockByNumber(*endNum)
		if endBlock == nil {
			return nil, fmt.Errorf("end block %d not found", *endNum)
		}
	}
	return api.getModifiedAccounts(startBlock, endBlock)
}

// GetModifiedAccountsByHash returns all accounts that have changed between the
// two blocks specified. A change is defined as a difference in nonce, balance,
// code hash, or storage hash.
//
// With one parameter, returns the list of accounts modified in the specified block.
func (api *PrivateDebugAPI) GetModifiedAccountsByHash(startHash common.Hash, endHash *common.Hash) ([]common.Address, error) {
	var startBlock, endBlock *types.Block
	startBlock = api.eth.blockchain.GetBlockByHash(startHash)
	if startBlock == nil {
		return nil, fmt.Errorf("start block %x not found", startHash)
	}

	if endHash == nil {
		endBlock = startBlock
		startBlock = api.eth.blockchain.GetBlockByHash(startBlock.ParentHash())
		if startBlock == nil {
			return nil, fmt.Errorf("block %x has no parent", endBlock.Number())
		}
	} else {
		endBlock = api.eth.blockchain.GetBlockByHash(*endHash)
		if endBlock == nil {
			return nil, fmt.Errorf("end block %x not found", *endHash)
		}
	}
	return api.getModifiedAccounts(startBlock, endBlock)
}

func (api *PrivateDebugAPI) getModifiedAccounts(startBlock, endBlock *types.Block) ([]common.Address, error) {
	if startBlock.Number().Uint64() >= endBlock.Number().Uint64() {
		return nil, fmt.Errorf("start block height (%d) must be less than end block height (%d)", startBlock.Number().Uint64(), endBlock.Number().Uint64())
	}

	oldTrie, err := trie.NewSecure(startBlock.Root(), trie.NewDatabase(api.eth.chainDb), 0)
	if err != nil {
		return nil, err
	}
	newTrie, err := trie.NewSecure(endBlock.Root(), trie.NewDatabase(api.eth.chainDb), 0)
	if err != nil {
		return nil, err
	}

	diff, _ := trie.NewDifferenceIterator(oldTrie.NodeIterator([]byte{}), newTrie.NodeIterator([]byte{}))
	iter := trie.NewIterator(diff)

	var dirty []common.Address
	for iter.Next() {
		key := newTrie.GetKey(iter.Key)
		if key == nil {
			return nil, fmt.Errorf("no preimage found for hash %x", iter.Key)
		}
		dirty = append(dirty, common.BytesToAddress(key))
	}
	return dirty, nil
}

func (api *PublicEthereumAPI) NewSideChain(appId string) error {
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: false}
		cacheConfig = &core.CacheConfig{
			Disabled:      false,
			TrieNodeLimit: DefaultConfig.TrieCache,
			TrieTimeLimit: DefaultConfig.TrieTimeout}
	)
	// 取得chainConfig
	stored := rawdb.ReadCanonicalHash(api.e.chainDb, 0, appId)
	if (stored == common.Hash{}) {
		return fmt.Errorf("wrong appId , appId %s does not exist in blockChain", appId)
	}
	config := rawdb.ReadChainConfig(api.e.chainDb, stored, appId)
	if _, ok := api.e.sideChains[appId]; ok {
		log.Info("不能重复引入侧链", "appId", appId)
		return nil
	}
	// engine
	engine := MakeSideEngine(api.e, config, api.e.chainDb)
	// blockChain
	blockChain, bcErr := core.NewBlockChain(api.e.chainDb, cacheConfig, config, engine, vmConfig)
	if bcErr != nil {
		return bcErr
	}
	api.e.sideChains[config.AppId] = blockChain
	// tx_pool
	sideTxPool := core.NewTxPool(core.DefaultTxPoolConfig, config, blockChain, api.e)
	api.e.sideTxPool[config.AppId] = sideTxPool
	//Miner
	sideMiner := miner.New(api.e, config, api.e.EventMux(), blockChain.Engine(), appId)
	api.e.sideMiner[appId] = sideMiner
	// downloader
	dl := downloader.New(DefaultConfig.SyncMode, api.e.chainDb, api.e.protocolManager.eventMux, blockChain, nil, api.e.protocolManager.removePeer)
	api.e.protocolManager.SideDownloader[appId] = dl
	initDownloader(appId, api.e)
	go api.e.protocolManager.syncer(appId)

	log.Info("您已引入的应用链数量", "num", len(api.e.sideChains))
	for _, b := range api.e.sideChains {
		log.Info("应用链", "appId", b.Config().AppId)
	}
	return nil
}

func initDownloader(appId string, eth *Ethereum) {
	if DefaultConfig.SyncMode == downloader.FastSync && eth.sideChains[appId].CurrentBlock().NumberU64() > 0 {
		log.Warn("blockChain not empty, fast sync disabled", "appId", appId)
		DefaultConfig.SyncMode = downloader.FullSync
	}
	if DefaultConfig.SyncMode == downloader.FastSync {
		eth.protocolManager.fastSync[appId] = uint32(1)
	}
	eth.protocolManager.noMorePeers[appId] = make(chan struct{})
	peers := eth.protocolManager.downloader.GetPeers().AllPeers()
	for _, peer := range peers {
		eth.protocolManager.SideDownloader[appId].RegisterPeer(peer.GetId(), peer.GetVersion(), peer.GetPeer())
	}
}

func MakeSideEngine(s *Ethereum, chainConfig *params.ChainConfig, db ethdb.Database) consensus.Engine {

	return nil
}

func (api *PrivateMinerAPI) SideMinerStart(appId string) error {
	chain := api.e.sideChains[appId]
	if chain == nil {
		return fmt.Errorf("the side chain is not created. Please create the side chain by command 'eth.NewSideChain(appId)'")
	}
	if sideMiner, ok := api.e.sideMiner[appId]; sideMiner == nil || !ok {
		api.e.sideMiner[appId] = miner.New(api.e, chain.Config(), api.e.EventMux(), chain.Engine(), appId)
	}
	api.e.sideMiner[appId].SetIsPassive(false)
	api.e.StartMining(true, appId)
	return nil
}

func (api *PublicEthereumAPI) DeleteSideChain(appId string) error {
	if _, ok := api.e.SideBlockChain(appId); !ok {
		log.Error("the side chain is not created. Please create the side chain by command 'eth.NewSideChain(appId)'")
		return nil
	}
	if miner, ok := api.e.sideMiner[appId]; miner != nil && ok {
		miner.Stop()
	}
	if chain, ok := api.e.sideChains[appId]; chain != nil && ok {
		chain.Stop()
	}
	if txPool, ok := api.e.sideTxPool[appId]; txPool != nil && ok {
		txPool.Stop()
	}
	api.e.protocolManager.noMorePeers[appId] <- struct{}{}
	delete(api.e.sideMiner, appId)
	delete(api.e.sideChains, appId)
	delete(api.e.sideTxPool, appId)
	delete(api.e.protocolManager.SideDownloader, appId)
	delete(api.e.protocolManager.noMorePeers, appId)
	return nil
}

func (api *PublicEthereumAPI) GetAllApp() map[string]*params.ChainConfig {
	rMap := rawdb.ReadAllChainConfig(api.e.chainDb)
	delete(rMap, "")
	return rMap
}

func (api *PublicEthereumAPI) GetBlockRewards(blockNumber uint64, appId string) (map[common.Address]string, error) {
	var allRewards = make(map[common.Address]string)


	return allRewards, nil
}

func (api *PublicEthereumAPI) GetContractAddrByAppId(appId string) common.Address{
	stored := rawdb.ReadCanonicalHash(api.e.chainDb, 0, appId)
	config := rawdb.ReadChainConfig(api.e.chainDb, stored, appId)
	return crypto.CreateAddress(config.Author, 1, config.AppId)
}
