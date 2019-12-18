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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/consensus"
	"github.com/CarLiveChainCo/goiov/core"
	"github.com/CarLiveChainCo/goiov/core/rawdb"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/eth/downloader"
	"github.com/CarLiveChainCo/goiov/eth/fetcher"
	"github.com/CarLiveChainCo/goiov/ethdb"
	"github.com/CarLiveChainCo/goiov/event"
	"github.com/CarLiveChainCo/goiov/log"
	"github.com/CarLiveChainCo/goiov/p2p"
	"github.com/CarLiveChainCo/goiov/p2p/discover"
	"github.com/CarLiveChainCo/goiov/params"
	"github.com/CarLiveChainCo/goiov/rlp"
)

const (
	// 目标返回块、头或节点数据的最大大小。
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	// RLP编码的块头的近似大小
	estHeaderRlpSize = 500 // Approximate size of an RLP encoded block header

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	// txChanSize是监听NewTxsEvent的通道的大小。
	// 这个数字是从tx池的大小引用的。
	txChanSize = 4096
)

var (
	// 允许节点响应DAO握手挑战的时间
	daoChallengeTimeout = 15 * time.Second // Time allowance for a node to reply to the DAO handshake challenge
)

// errIncompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
// 如果请求的协议和配置不兼容(低协议版本限制和高要求)，则返回errIncompatibleConfig。
var errIncompatibleConfig = errors.New("incompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type ProtocolManager struct {
	networkId uint64

	// 标记是否启用了快速同步(如果已经有块，则禁用)
	fastSync map[string]uint32 // Flag whether fast sync is enabled (gets disabled if we already have blocks)
	// 标记我们是否被认为是同步的(启用事务处理)
	acceptTxs uint32 // Flag whether we're considered synchronised (enables transaction processing)

	txpool      txPool
	blockchain  *core.BlockChain
	chainconfig *params.ChainConfig
	maxPeers    int

	downloader *downloader.Downloader
	fetcher    *fetcher.Fetcher
	peers      *peerSet

	SubProtocols []p2p.Protocol

	eventMux      *event.TypeMux
	txsCh         chan core.NewTxsEvent
	txsSub        event.Subscription
	minedBlockSub *event.TypeMuxSubscription

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	txsyncCh    chan *txsync
	quitSync    chan struct{}
	noMorePeers map[string]chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	// 等待组用于在下载和处理期间优雅地关闭
	wg sync.WaitGroup

	// 将侧链引入
	SideChains     map[string]*core.BlockChain //已经引入的链
	SideDownloader map[string]*downloader.Downloader
	SideTxPool     map[string]*core.TxPool
}

// NewProtocolManager returns a new Ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the Ethereum network.
// 返回一个新的Ethereum子协议管理器。Ethereum子协议管理具有Ethereum网络功能的对等节点。
func NewProtocolManager(config *params.ChainConfig, mode downloader.SyncMode, networkId uint64, mux *event.TypeMux, txpool txPool, engine consensus.Engine, blockchain *core.BlockChain, chaindb ethdb.Database, sideChains map[string]*core.BlockChain, sideTxPool map[string]*core.TxPool) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	// 使用基本字段创建协议管理器
	manager := &ProtocolManager{
		networkId:   networkId,
		eventMux:    mux,
		txpool:      txpool,
		blockchain:  blockchain,
		chainconfig: config,
		SideChains:  sideChains,
		SideTxPool:  sideTxPool,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer),
		noMorePeers: make(map[string]chan struct{}),
		txsyncCh:    make(chan *txsync),
		quitSync:    make(chan struct{}),
		fastSync:    make(map[string]uint32),
	}
	manager.noMorePeers[""] = make(chan struct{})
	for id := range sideChains {
		manager.noMorePeers[id] = make(chan struct{})
	}
	// Figure out whether to allow fast sync or not
	// 弄清楚是否允许快速同步
	if mode == downloader.FastSync && blockchain.CurrentBlock().NumberU64() > 0 {
		log.Warn("Blockchain not empty, fast sync disabled")
		mode = downloader.FullSync
	}
	if mode == downloader.FastSync {
		manager.fastSync[""] = uint32(1)
	}
	for id, chain := range sideChains {
		if mode == downloader.FastSync && chain.CurrentBlock().NumberU64() > 0 {
			log.Warn("Blockchain not empty, fast sync disabled")
			mode = downloader.FullSync
		}
		if mode == downloader.FastSync {
			manager.fastSync[id] = uint32(1)
		}
	}
	// Initiate a sub-protocol for every implemented version we can handle
	// 为我们能够处理的每个实现版本初始化一个子协议
	manager.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Skip protocol version if incompatible with the mode of operation
		// 跳过与协议版本不兼容的操作模式
		if mode == downloader.FastSync && version < eth63 {
			continue
		}
		// Compatible; initialise the sub-protocol
		// 如果兼容：初始化sub-protocol
		version := version // Closure for the run	// 运行的闭包
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), p, rw)
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					return manager.handle(peer)
				case <-manager.quitSync:
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return manager.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := manager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errIncompatibleConfig
	}
	// Construct the different synchronisation mechanisms
	// 构造不同的同步机制
	manager.downloader = downloader.New(mode, chaindb, manager.eventMux, blockchain, nil, manager.removePeer)
	manager.SideDownloader = make(map[string]*downloader.Downloader)
	for id, chain := range manager.SideChains {
		manager.SideDownloader[id] = downloader.New(mode, chaindb, manager.eventMux, chain, nil, manager.removePeer)
	}
	validator := func(header *types.Header) error {
		if chain, err := manager.ExistsAppId(header.Appid); err != nil {
			return nil
		} else {
			return chain.Engine().VerifyHeader(chain, header, true)
		}
	}
	heighter := func(appId string) int64 {
		if chain, err := manager.ExistsAppId(appId); err != nil {
			return -1
		} else {
			return int64(chain.CurrentBlock().NumberU64())
		}
	}
	inserter := func(blocks types.Blocks) (int, error) {
		// If fast sync is running, deny importing weird blocks
		// 如果正在运行快速同步，则拒绝导入奇怪的块
		fs := manager.fastSync[blocks[0].Header().Appid]
		if atomic.LoadUint32(&fs) == 1 {
			log.Warn("Discarded bad propagated block", "number", blocks[0].Number(), "hash", blocks[0].Hash())
			return 0, nil
		}
		atomic.StoreUint32(&manager.acceptTxs, 1) // Mark initial sync done on any fetcher import // 在任何获取器导入上标记完成的初始同步
		// 只有一个地方使用inserter，并且只传入了一个块，所以仅判断第一个就行了
		if chain, err := manager.ExistsAppId(blocks[0].Header().Appid); err != nil {
			return 0, nil
		} else {
			return chain.InsertChain(blocks)
		}
	}
	manager.fetcher = fetcher.New(manager.GetBlockByHashAll, validator, manager.BroadcastBlock, heighter, inserter, manager.removePeer, manager.blockchain)

	return manager, nil
}

func (pm *ProtocolManager) GetBlockByHashAll(hash common.Hash, appId string) *types.Block {
	if chain, err := pm.ExistsAppId(appId); err != nil {
		return nil
	} else {
		return chain.GetBlockByHash(hash)
	}
}

func (pm *ProtocolManager) removePeer(id string) {
	// joker
	// Short circuit if the peer was already removed
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing Ethereum peer", "peer", id)

	// Unregister the peer from the downloader and Ethereum peer set
	pm.downloader.UnregisterPeer(id)
	for _, dl := range pm.SideDownloader {
		dl.UnregisterPeer(id)
	}
	if err := pm.peers.Unregister(id); err != nil {
		log.Error("Peer removal failed", "peer", id, "err", err)
	}

	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (pm *ProtocolManager) Start(maxPeers int) {
	pm.maxPeers = maxPeers

	// broadcast transactions
	pm.txsCh = make(chan core.NewTxsEvent, txChanSize)
	pm.txsSub = pm.txpool.SubscribeNewTxsEvent(pm.txsCh)
	go pm.txBroadcastLoop()

	// broadcast mined blocks
	pm.minedBlockSub = pm.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go pm.minedBroadcastLoop()
	// start sync handlers
	go pm.syncer("")
	for id := range pm.SideChains {
		go pm.syncer(id)
	}
	go pm.txsyncLoop()
}

func (pm *ProtocolManager) Stop() {
	log.Info("Stopping Ethereum protocol")

	pm.txsSub.Unsubscribe()        // quits txBroadcastLoop
	pm.minedBlockSub.Unsubscribe() // quits blockBroadcastLoop
	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	for _, p := range pm.noMorePeers {
		p <- struct{}{}
	}
	// Quit fetcher, txsyncLoop.
	close(pm.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	pm.peers.Close()

	// Wait for all peer handler goroutines and the loops to come down.
	pm.wg.Wait()

	log.Info("Ethereum protocol stopped")
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
// 句柄是用来管理eth对等点的生命周期的回调。当此函数终止时，对等点断开连接。
func (pm *ProtocolManager) handle(p *peer) error {
	// Ignore maxPeers if this is a trusted peer
	// 如果这是一个受信任的对等点，则忽略maxpeer
	if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("Ethereum peer connected", "name", p.Name())

	// Execute the Ethereum handshake
	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
		hash    = head.Hash()
		number  = head.Number.Uint64()
		td      = pm.blockchain.GetTd(hash, number)
	)
	if err := p.Handshake(pm.networkId, td, hash, genesis.Hash()); err != nil {
		p.Log().Debug("Ethereum handshake failed", "err", err)
		return err
	}
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("Ethereum peer registration failed", "err", err)
		return err
	}
	defer pm.removePeer(p.id)

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	// 在下载器中注册对等点。如果下载器认为它是禁止的，我们断开连接
	if err := pm.downloader.RegisterPeer(p.id, p.version, p); err != nil {
		return err
	}
	for _, dl := range pm.SideDownloader {
		if err := dl.RegisterPeer(p.id, p.version, p); err != nil {
			return err
		}
	}
	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	// 现有传播事务。在此之后出现的新事务将通过广播发送。
	pm.syncTransactions(p)

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("Ethereum message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	// Handle the message depending on its contents
	switch {
	case msg.Code == StatusMsg:
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

		// Block header query, collect the requested headers and reply
		// head查询
	case UnJointToMsg(msg.Code) == GetBlockHeadersMsg:
		// Decode the complex header query
		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		hashMode := query.Origin.Hash != (common.Hash{})
		chain, err := pm.ExistsAppId(UnJointToAppId(msg.Code))
		if err != nil {
			return nil
		}
		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
			// Retrieve the next header satisfying the query
			var origin *types.Header
			if hashMode {
				origin = chain.GetHeaderByHash(query.Origin.Hash)
			} else {
				origin = chain.GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			number := origin.Number.Uint64()
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next header of the query
			switch {
			case query.Origin.Hash != (common.Hash{}) && query.Reverse:
				// Hash based traversal towards the genesis block
				for i := 0; i < int(query.Skip)+1; i++ {
					if header := chain.GetHeader(query.Origin.Hash, number); header != nil {
						query.Origin.Hash = header.ParentHash
						number--
					} else {
						unknown = true
						break
					}
				}
			case query.Origin.Hash != (common.Hash{}) && !query.Reverse:
				// Hash based traversal towards the leaf block
				var (
					current = origin.Number.Uint64()
					next    = current + query.Skip + 1
				)
				if next <= current {
					infos, _ := json.MarshalIndent(p.Peer.Info(), "", "  ")
					p.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
					unknown = true
				} else {
					if header := chain.GetHeaderByNumber(next); header != nil {
						if chain.GetBlockHashesFromHash(header.Hash(), query.Skip+1)[query.Skip] == query.Origin.Hash {
							query.Origin.Hash = header.Hash()
						} else {
							unknown = true
						}
					} else {
						unknown = true
					}
				}
			case query.Reverse:
				// Number based traversal towards the genesis block
				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= query.Skip + 1
				} else {
					unknown = true
				}

			case !query.Reverse:
				// Number based traversal towards the leaf block
				query.Origin.Number += query.Skip + 1
			}
		}
		return p.SendBlockHeaders(headers, UnJointToAppId(msg.Code))
		// 不用appid分解
	case UnJointToMsg(msg.Code) == BlockHeadersMsg:
		// A batch of headers arrived to one of our previous requests
		var headers []*types.Header
		if err := msg.Decode(&headers); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		dl := pm.downloader
		if _, err := pm.ExistsAppId(UnJointToAppId(msg.Code)); err == nil {
			if UnJointToAppId(msg.Code) != "" {
				dl = pm.SideDownloader[UnJointToAppId(msg.Code)]
			}
		} else {
			return nil
		}
		// Filter out any explicitly requested headers, deliver the rest to the downloader
		filter := len(headers) == 1
		if filter {
			// Irrelevant of the fork checks, send the header to the fetcher just in case
			headers = pm.fetcher.FilterHeaders(p.id, headers, time.Now())
		}
		if len(headers) > 0 || !filter {
			err := dl.DeliverHeaders(p.id, headers)
			if err != nil {
				log.Debug("Failed to deliver headers", "err", err)
			}
		}

		// 获取bodies请求
	case UnJointToMsg(msg.Code) == GetBlockBodiesMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			hash   common.Hash
			bytes  int
			bodies []rlp.RawValue
		)
		chain, err := pm.ExistsAppId(UnJointToAppId(msg.Code))
		if err != nil {
			return nil
		}
		for bytes < softResponseLimit && len(bodies) < downloader.MaxBlockFetch {
			// Retrieve the hash of the next block
			// 检索下一个块的散列
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block body, stopping if enough was found
			// 检索请求的块体，如果找到足够的块，则停止
			if data := chain.GetBodyRLP(hash); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		return p.SendBlockBodiesRLP(bodies, UnJointToAppId(msg.Code))

		// 收到bodies
	case UnJointToMsg(msg.Code) == BlockBodiesMsg:
		// A batch of block bodies arrived to one of our previous requests
		var request blockBodiesData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		dl := pm.downloader
		if _, err := pm.ExistsAppId(UnJointToAppId(msg.Code)); err == nil {
			if UnJointToAppId(msg.Code) != "" {
				dl = pm.SideDownloader[UnJointToAppId(msg.Code)]
			}
		} else {
			return nil
		}
		// Deliver them all to the downloader for queuing
		transactions := make([][]*types.Transaction, len(request))
		uncles := make([][]*types.Header, len(request))

		for i, body := range request {
			transactions[i] = body.Transactions
			uncles[i] = body.Uncles
		}
		// Filter out any explicitly requested bodies, deliver the rest to the downloader
		filter := len(transactions) > 0 || len(uncles) > 0
		if filter {
			transactions, uncles = pm.fetcher.FilterBodies(p.id, transactions, uncles, time.Now())
		}
		if len(transactions) > 0 || len(uncles) > 0 || !filter {
			err := dl.DeliverBodies(p.id, transactions, uncles)
			if err != nil {
				log.Debug("Failed to deliver bodies", "err", err)
			}
		}

		// 根据hash发状态
	case p.version >= eth63 && UnJointToMsg(msg.Code) == GetNodeDataMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash  common.Hash
			bytes int
			data  [][]byte
		)
		chain := pm.blockchain
		if sideChain, err := pm.ExistsAppId(UnJointToAppId(msg.Code)); err != nil {
			return nil
		} else if UnJointToAppId(msg.Code) != "" {
			chain = sideChain
		}
		for bytes < softResponseLimit && len(data) < downloader.MaxStateFetch {
			// Retrieve the hash of the next state entry
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested state entry, stopping if enough was found
			if entry, err := chain.TrieNode(hash); err == nil {
				data = append(data, entry)
				bytes += len(entry)
			}
		}
		return p.SendNodeData(data, UnJointToAppId(msg.Code))

		// 拿状态
	case p.version >= eth63 && UnJointToMsg(msg.Code) == NodeDataMsg:
		// A batch of node state data arrived to one of our previous requests
		var data [][]byte
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		dl := pm.downloader
		if _, err := pm.ExistsAppId(UnJointToAppId(msg.Code)); err != nil {
			return nil
		} else if UnJointToAppId(msg.Code) != "" {
			dl = pm.SideDownloader[UnJointToAppId(msg.Code)]
		}
		// Deliver all to the downloader
		if err := dl.DeliverNodeData(p.id, data); err != nil {
			log.Debug("Failed to deliver node state data", "err", err)
		}

		//发账单
	case p.version >= eth63 && UnJointToMsg(msg.Code) == GetReceiptsMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash     common.Hash
			bytes    int
			receipts []rlp.RawValue
		)
		chain := pm.blockchain
		if sideChain, err := pm.ExistsAppId(UnJointToAppId(msg.Code)); err != nil {
			return nil
		} else if UnJointToAppId(msg.Code) != "" {
			chain = sideChain
		}
		for bytes < softResponseLimit && len(receipts) < downloader.MaxReceiptFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block's receipts, skipping if unknown to us
			results := chain.GetReceiptsByHash(hash)
			if results == nil {
				if header := chain.GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
					continue
				}
			}
			// If known, encode and queue for response packet
			if encoded, err := rlp.EncodeToBytes(results); err != nil {
				log.Error("Failed to encode receipt", "err", err)
			} else {
				receipts = append(receipts, encoded)
				bytes += len(encoded)
			}
		}
		return p.SendReceiptsRLP(receipts, UnJointToAppId(msg.Code))

		//收账单
	case p.version >= eth63 && UnJointToMsg(msg.Code) == ReceiptsMsg:
		// A batch of receipts arrived to one of our previous requests
		var receipts [][]*types.Receipt
		if err := msg.Decode(&receipts); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		dl := pm.downloader
		if _, err := pm.ExistsAppId(UnJointToAppId(msg.Code)); err != nil {
			return nil
		} else if UnJointToAppId(msg.Code) != "" {
			dl = pm.SideDownloader[UnJointToAppId(msg.Code)]
		}
		// Deliver all to the downloader
		if err := dl.DeliverReceipts(p.id, receipts); err != nil {
			log.Debug("Failed to deliver receipts", "err", err)
		}

		// 验证区块在本地是否存在，发送不通过验证的hash
	case UnJointToMsg(msg.Code) == NewBlockHashesMsg:
		var announces newBlockHashesData
		if err := msg.Decode(&announces); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		// Mark the hashes as present at the remote node
		for _, block := range announces {
			p.MarkBlock(block.Hash)
		}
		// Schedule all the unknown hashes for retrieval
		unknown := make(newBlockHashesData, 0, len(announces))

		for _, block := range announces {
			chain, err := pm.ExistsAppId(UnJointToAppId(msg.Code))
			if err != nil {
				return nil
			}
			if !chain.HasBlock(block.Hash, block.Number) {
				unknown = append(unknown, block)
			}
		}
		for _, block := range unknown {
			pm.fetcher.Notify(p.id, UnJointToAppId(msg.Code), block.Hash, block.Number, time.Now(), p.RequestOneHeader, p.RequestBodies)
		}

	case UnJointToMsg(msg.Code) == NewBlockMsg:
		// Retrieve and decode the propagated block
		// 检索和解码传播的块
		var request newBlockData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		chain, err := pm.ExistsAppId(UnJointToAppId(msg.Code))
		if err != nil {
			return nil
		}
		request.Block.ReceivedAt = msg.ReceivedAt
		request.Block.ReceivedFrom = p

		// Mark the peer as owning the block and schedule it for import
		// 将对等点标记为拥有块，并安排它进行导入
		p.MarkBlock(request.Block.Hash())
		pm.fetcher.Enqueue(p.id, request.Block)

		// Assuming the block is importable by the peer, but possibly not yet done so,
		// calculate the head hash and TD that the peer truly must have.
		// 假设数据块可以被对等方导入，但可能还没有导入，那么计算对等方真正需要的头哈希和TD。
		var (
			trueHead = request.Block.ParentHash()
			trueTD   = new(big.Int).Sub(request.TD, request.Block.Difficulty())
		)
		// Update the peers total difficulty if better than the previous
		// 如果比以前更好，则更新总难度
		if _, td := p.Head(UnJointToAppId(msg.Code)); trueTD.Cmp(td) > 0 {
			p.SetHead(trueHead, trueTD, UnJointToAppId(msg.Code))

			// Schedule a sync if above ours. Note, this will not fire a sync for a gap of
			// a singe block (as the true TD is below the propagated block), however this
			// scenario should easily be covered by the fetcher.
			currentBlock := chain.CurrentBlock()
			if trueTD.Cmp(chain.GetTd(currentBlock.Hash(), currentBlock.NumberU64())) > 0 {
				go pm.synchronise(p, UnJointToAppId(msg.Code))
			}
		}

	case msg.Code == TxMsg:
		// Transactions arrived, make sure we have a valid and fresh chain to handle them
		if atomic.LoadUint32(&pm.acceptTxs) == 0 {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		idTx := make(map[string][]*types.Transaction)
		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			pm.TxShunt(idTx , tx)
			p.MarkTransaction(tx.Hash())
		}
		for id, txs := range idTx {
			if id == "noChain"{
				if len(txs) > 0 {
					pm.BroadcastTxs(txs)
				}
			}else if id == "main"{
				pm.txpool.AddRemotes(txs)
			}else{
				pm.SideTxPool[id].AddRemotes(txs)
			}
		}
	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

func (pm *ProtocolManager) TxShunt(idTx map[string][]*types.Transaction , tx *types.Transaction){
	store := rawdb.ReadCanonicalHash(pm.blockchain.Getdb(), 0, tx.AppId())
	sideTx := tx.AppId() != "" && tx.To() != nil
	addContract := store != common.Hash{} && tx.AppId() != "" && tx.To() == nil
	if sideTx || addContract {
		//noChain为没有引入的链，仅广播
		if sideTxPool, ok := pm.SideTxPool[tx.AppId()]; sideTxPool == nil || !ok {
			idTx["noChain"] = append(idTx["noChain"], tx)
		} else {
			idTx[tx.AppId()] = append(idTx[tx.AppId()], tx)
		}
	} else {
		idTx["main"] = append(idTx["main"], tx)
	}
}

// BroadcastBlock will either propagate a block to a subset of it's peers, or
// will only announce it's availability (depending what's requested).
func (pm *ProtocolManager) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := pm.peers.PeersWithoutBlock(hash)
	chain, err := pm.ExistsAppId(block.Header().Appid)
	if err != nil {
		return
	}
	// If propagation is requested, send to a subset of the peer
	if propagate {
		// Calculate the TD of the block (it's not imported yet, so block.Td is not valid)
		var td *big.Int
		if parent := chain.GetBlock(block.ParentHash(), block.NumberU64()-1); parent != nil {
			td = new(big.Int).Add(block.Difficulty(), chain.GetTd(block.ParentHash(), block.NumberU64()-1))
		} else {
			log.Error("Propagating dangling block", "number", block.Number(), "hash", hash)
			return
		}
		// Send the block to a subset of our peers
		transfer := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range transfer {
			peer.AsyncSendNewBlock(block, td)
		}
		log.Trace("Propagated block", "hash", hash, "recipients", len(transfer), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
		return
	}
	// Otherwise if the block is indeed in out own chain, announce it
	if chain.HasBlock(hash, block.NumberU64()) {
		for _, peer := range peers {
			// joker
			peer.AsyncSendNewBlockHash(block)
		}
		log.Trace("Announced block", "hash", hash, "recipients", len(peers), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
	}
}

// BroadcastTxs will propagate a batch of transactions to all peers which are not known to
// already have the given transaction.
func (pm *ProtocolManager) BroadcastTxs(txs types.Transactions) {
	var txset = make(map[*peer]types.Transactions)

	// Broadcast transactions to a batch of peers not knowing about it
	for _, tx := range txs {
		peers := pm.peers.PeersWithoutTx(tx.Hash())
		for _, peer := range peers {
			txset[peer] = append(txset[peer], tx)
		}
		log.Trace("Broadcast transaction", "hash", tx.Hash(), "recipients", len(peers))
	}
	// FIXME include this again: peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for peer, txs := range txset {
		peer.AsyncSendTransactions(txs)
	}
}

// Mined broadcast loop
func (pm *ProtocolManager) minedBroadcastLoop() {
	//log.Info("---打开区块广播循环", "chainAppId", pm.chainconfig.AppId)
	// automatically stops if unsubscribe
	for obj := range pm.minedBlockSub.Chan() {
		switch ev := obj.Data.(type) {
		case core.NewMinedBlockEvent:
			pm.BroadcastBlock(ev.Block, true)  // First propagate block to peers
			pm.BroadcastBlock(ev.Block, false) // Only then announce to the rest
		}
	}
}

func (pm *ProtocolManager) txBroadcastLoop() {
	//log.Info("---打开交易广播循环", "chainAppId", pm.chainconfig.AppId)
	for {
		select {
		case event := <-pm.txsCh:
			pm.BroadcastTxs(event.Txs)

			// Err() channel will be closed when unsubscribing.
		case <-pm.txsSub.Err():
			return
		}
	}
}

// NodeInfo represents a short summary of the Ethereum sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network    uint64              `json:"network"`    // Ethereum network ID (1=Frontier, 2=Morden, Ropsten=3, Rinkeby=4)
	Difficulty *big.Int            `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    common.Hash         `json:"genesis"`    // SHA3 hash of the host's genesis block
	Config     *params.ChainConfig `json:"config"`     // Chain configuration for the fork rules
	Head       common.Hash         `json:"head"`       // SHA3 hash of the host's best owned block
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (pm *ProtocolManager) NodeInfo() *NodeInfo {
	currentBlock := pm.blockchain.CurrentBlock()
	return &NodeInfo{
		Network:    pm.networkId,
		Difficulty: pm.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64()),
		Genesis:    pm.blockchain.Genesis().Hash(),
		Config:     pm.blockchain.Config(),
		Head:       currentBlock.Hash(),
	}
}

func (pm *ProtocolManager) ExistsAppId(appId string) (*core.BlockChain, error) {
	if appId != "" {
		if chain, ok := pm.SideChains[appId]; chain == nil || !ok {
			return nil, ErrNoSideChain
		} else {
			return chain, nil
		}
	}
	return pm.blockchain, nil
}
