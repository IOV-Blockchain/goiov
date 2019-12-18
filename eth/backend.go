// Copyright 2014 The go-ethereum Authors
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

// Package eth implements the Ethereum protocol.
package eth

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/CarLiveChainCo/goiov/accounts"
	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/common/hexutil"
	"github.com/CarLiveChainCo/goiov/consensus"

	"github.com/CarLiveChainCo/goiov/consensus/clique"
	"github.com/CarLiveChainCo/goiov/consensus/ethash"
	"github.com/CarLiveChainCo/goiov/core"
	"github.com/CarLiveChainCo/goiov/core/bloombits"
	"github.com/CarLiveChainCo/goiov/core/rawdb"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/core/vm"
	"github.com/CarLiveChainCo/goiov/eth/downloader"
	"github.com/CarLiveChainCo/goiov/eth/filters"
	"github.com/CarLiveChainCo/goiov/eth/gasprice"
	"github.com/CarLiveChainCo/goiov/ethdb"
	"github.com/CarLiveChainCo/goiov/event"
	"github.com/CarLiveChainCo/goiov/internal/ethapi"
	"github.com/CarLiveChainCo/goiov/log"
	"github.com/CarLiveChainCo/goiov/miner"
	"github.com/CarLiveChainCo/goiov/node"
	"github.com/CarLiveChainCo/goiov/p2p"
	"github.com/CarLiveChainCo/goiov/params"
	"github.com/CarLiveChainCo/goiov/rlp"
	"github.com/CarLiveChainCo/goiov/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Ethereum implements the Ethereum full node service.
type Ethereum struct {
	config      *Config
	chainConfig *params.ChainConfig
	nodeConfig  *node.Config

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Ethereum

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb ethdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *EthAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	etherbase common.Address

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)

	sideChains map[string]*core.BlockChain //已引入的链
	sideTxPool map[string]*core.TxPool     //已引入链的交易池
	sideMiner  map[string]*miner.Miner     //已引入链的矿工
}

func (s *Ethereum) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Ethereum object (including the
// initialisation of the common Ethereum object)
func New(ctx *node.ServiceContext, config *Config, nodeCfg *node.Config) (*Ethereum, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run eth.Ethereum in light sync mode, use les.LightEthereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	// 创建数据库
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	// 创建创世块，statedb，并写入数据库
	testFlag := (config.NetworkId != 1)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis, testFlag)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	eth := &Ethereum{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.Ethash, chainConfig, chainDb, testFlag),
		shutdownChan:  make(chan bool),
		networkId:     config.NetworkId,
		gasPrice:      config.GasPrice,
		etherbase:     config.Etherbase,
		bloomRequests: make(chan chan *bloombits.Retrieval),
		bloomIndexer:  NewBloomIndexer(chainDb, params.BloomBitsBlocks),
		nodeConfig:    nodeCfg,
		sideChains:    make(map[string]*core.BlockChain),
		sideMiner:     make(map[string]*miner.Miner),
		sideTxPool:    make(map[string]*core.TxPool),
	}

	log.Info("Initialising Ethereum protocol", "versions", ProtocolVersions, "network", config.NetworkId)
	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geth upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	// 创建一条链
	eth.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, eth.chainConfig, eth.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eth.blockchain.SetHead(compat.RewindTo)
		rMap := rawdb.ReadAllChainConfig(chainDb)
		rMap[""] = chainConfig
		rawdb.WriteChainConfig(chainDb, genesisHash, rMap)
	}
	eth.bloomIndexer.Start(eth.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	eth.txPool = core.NewTxPool(config.TxPool, eth.chainConfig, eth.blockchain, eth)
	if eth.protocolManager, err = NewProtocolManager(eth.chainConfig, config.SyncMode, config.NetworkId, eth.eventMux, eth.txPool, eth.engine, eth.blockchain, chainDb, eth.sideChains, eth.sideTxPool); err != nil {
		return nil, err
	}
	var appId = rawdb.ReadAppId(eth.chainDb)
	if len(appId) != 0 {
		for _, id := range appId {
			eth.NewSideChain(false, id)
		}
	}
	log.Info("已引入侧链数", "num", len(eth.sideChains))
	eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine, "")
	eth.miner.SetExtra(makeExtraData(config.ExtraData))

	eth.APIBackend = &EthAPIBackend{eth, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	eth.APIBackend.gpo = gasprice.NewOracle(eth.APIBackend, gpoParams)

	return eth, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"geth",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (ethdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*ethdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine(ctx, &config.Ethash, chainConfig, chainDb, testFlag),
// CreateConsensusEngine creates the required type of consensus engine instance for an Ethereum service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ethash.Config, chainConfig *params.ChainConfig, db ethdb.Database, testFlag ...bool) consensus.Engine {
	//func CreateConsensusEngine(ctx *node.ServiceContext, chainConfig *params.ChainConfig, db ethdb.Database, testFlag ...bool) consensus.Engine {
	// If proof-of-authority is requested, set it up

	return nil
}

// APIs return the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Ethereum) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Ethereum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Ethereum) Etherbase() (eb common.Address, err error) {
	s.lock.RLock()
	etherbase := s.etherbase
	s.lock.RUnlock()

	if etherbase != (common.Address{}) {
		log.Info("Ethereum Etherbase return directly")
		return etherbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etherbase := accounts[0].Address

			s.lock.Lock()
			s.etherbase = etherbase
			s.lock.Unlock()

			log.Info("Etherbase automatically configured", "address", etherbase)
			return etherbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("etherbase must be explicitly specified")
}

// SetEtherbase sets the mining reward address.
func (s *Ethereum) SetEtherbase(etherbase common.Address) {
	log.Info("Ethereum SetEtherbase", "etherbase", etherbase)
	s.lock.Lock()
	s.etherbase = etherbase
	s.lock.Unlock()

	s.miner.SetEtherbase(etherbase)
}

func (s *Ethereum) StartMining(local bool, id string) error {
	eb, err := s.Etherbase()
	if err != nil {
		log.Error("Cannot start mining without etherbase", "err", err)
		return fmt.Errorf("etherbase missing: %v", err)
	}
	log.Info("Ethereum StartMining", "Etherbase", eb)

	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Etherbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}

	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so none will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	if id != "" {
		log.Info("---取出侧链准备挖矿---", "appId", id)
		go s.sideMiner[id].Start(eb, id)
	} else {
		go s.miner.Start(eb, id)
	}

	return nil
}

func (s *Ethereum) StopMining()         { s.miner.Stop() }
func (s *Ethereum) IsMining() bool      { return s.miner.Mining() }
func (s *Ethereum) Miner() *miner.Miner { return s.miner }

func (s *Ethereum) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Ethereum) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Ethereum) TxPool() *core.TxPool               { return s.txPool }
func (s *Ethereum) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Ethereum) Engine() consensus.Engine           { return s.engine }
func (s *Ethereum) ChainDb() ethdb.Database            { return s.chainDb }
func (s *Ethereum) IsListening() bool                  { return true } // Always listening
func (s *Ethereum) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Ethereum) NetVersion() uint64                 { return s.networkId }
func (s *Ethereum) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *Ethereum) SideBlockChain(appId string) (*core.BlockChain, bool) {
	if appId == "" {
		return s.blockchain, true
	}
	if block, ok := s.sideChains[appId]; !ok {
		return nil, false
	} else {
		return block, true
	}
}
func (s *Ethereum) SideTxPool(appId string) *core.TxPool {
	if appId == "" {
		return s.txPool
	}
	if txPool, ok := s.sideTxPool[appId]; !ok {
		return nil
	} else {
		return txPool
	}
}
func (s *Ethereum) SideMiner(appId string) core.Miner {
	if appId == "" {
		return s.miner
	}
	if sMiner, ok := s.sideMiner[appId]; !ok {
		return nil
	} else {
		return sMiner
	}
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Ethereum) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Ethereum protocol implementation.
func (s *Ethereum) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

func (s *Ethereum) NewSideChain(idSync bool, appId string) error {
	if _, ok := s.sideChains[appId]; ok {
		return nil
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: false}
		cacheConfig = &core.CacheConfig{
			Disabled:      false,
			TrieNodeLimit: DefaultConfig.TrieCache,
			TrieTimeLimit: DefaultConfig.TrieTimeout}
	)
	// 取得chainConfig
	stored := rawdb.ReadCanonicalHash(s.chainDb, 0, appId)
	if (stored == common.Hash{}) {
		return fmt.Errorf("appId %s does not exist", appId)
	}
	config := rawdb.ReadChainConfig(s.chainDb, stored, appId)
	// engine
	engine := MakeSideEngine(s, config, s.chainDb)
	// blockChain
	blockChain, bcErr := core.NewBlockChain(s.chainDb, cacheConfig, config, engine, vmConfig)
	if bcErr != nil {
		return bcErr
	}
	s.sideChains[config.AppId] = blockChain
	// tx_pool
	sideTxPool := core.NewTxPool(core.DefaultTxPoolConfig, config, blockChain, s)
	s.sideTxPool[config.AppId] = sideTxPool
	// miner
	sideMiner := miner.New(s, config, s.EventMux(), blockChain.Engine(), appId)
	s.sideMiner[appId] = sideMiner
	// downloader
	dl := downloader.New(DefaultConfig.SyncMode, s.chainDb, s.protocolManager.eventMux, blockChain, nil, s.protocolManager.removePeer)
	s.protocolManager.SideDownloader[appId] = dl
	if idSync {
		go s.protocolManager.syncer(appId)
	}
	initDownloader(appId, s)

	log.Info("已引入侧链", "appId", appId)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Ethereum protocol.
func (s *Ethereum) Stop() error {
	var appId []string
	for id, _ := range s.sideChains {
		appId = append(appId, id)
	}
	rawdb.WriteAppId(s.chainDb, appId)
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	for _, sideChain := range s.sideChains {
		if sideChain != nil {
			sideChain.Stop()
		}
	}
	s.protocolManager.Stop()

	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	for _, sideTxPool := range s.sideTxPool {
		if sideTxPool != nil {
			sideTxPool.Stop()
		}
	}
	s.miner.Stop()
	for _, sideMiner := range s.sideMiner {
		if sideMiner != nil {
			sideMiner.Stop()
		}
	}
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}

func (s *Ethereum) NewPassiveMiner(id string) {
	chain, ok := s.sideChains[id]
	if !ok {
		return
	}
	if sm, ok := s.sideMiner[id]; sm == nil || !ok {
		s.sideMiner[id] = miner.New(s, chain.Config(), s.EventMux(), chain.Engine(), id)
	}
	s.sideMiner[id].SetIsPassive(true)
	s.StartMining(true, id)
}
func (s *Ethereum) GetSideChains() map[string]*core.BlockChain {
	return s.sideChains
}
