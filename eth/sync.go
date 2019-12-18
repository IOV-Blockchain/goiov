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
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/eth/downloader"
	"github.com/CarLiveChainCo/goiov/log"
	"github.com/CarLiveChainCo/goiov/p2p/discover"
)

const (
	forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing

	// This is the target size for the packs of transactions sent by txsyncLoop.
	// A pack can get larger than this if a single transactions exceeds this size.
	txsyncPackSize = 100 * 1024
)

type txsync struct {
	p   *peer
	txs []*types.Transaction
}

// syncTransactions starts sending all currently pending transactions to the given peer.
func (pm *ProtocolManager) syncTransactions(p *peer) {
	var txs types.Transactions
	//pending, _ := pm.txpool.Pending()
	pending, _ := pm.txpool.Pending()
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	if len(txs) == 0 {
		return
	}
	select {
	case pm.txsyncCh <- &txsync{p, txs}:
	case <-pm.quitSync:
	}
}

// txsyncLoop takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (pm *ProtocolManager) txsyncLoop() {
	//log.Info("---打开交易同步循环", "chainAppId", pm.chainconfig.AppId)
	var (
		pending = make(map[discover.NodeID]*txsync)
		sending = false               // whether a send is active
		pack    = new(txsync)         // the pack that is being sent
		done    = make(chan error, 1) // result of the send
	)

	// send starts a sending a pack of transactions from the sync.
	send := func(s *txsync) {
		// Fill pack with transactions up to the target size.
		size := common.StorageSize(0)
		pack.p = s.p
		pack.txs = pack.txs[:0]
		for i := 0; i < len(s.txs) && size < txsyncPackSize; i++ {
			pack.txs = append(pack.txs, s.txs[i])
			size += s.txs[i].Size()
		}
		// Remove the transactions that will be sent.
		s.txs = s.txs[:copy(s.txs, s.txs[len(pack.txs):])]
		if len(s.txs) == 0 {
			delete(pending, s.p.ID())
		}
		// Send the pack in the background.
		s.p.Log().Trace("Sending batch of transactions", "count", len(pack.txs), "bytes", size)
		sending = true
		go func() { done <- pack.p.SendTransactions(pack.txs) }()
	}

	// pick chooses the next pending sync.
	pick := func() *txsync {
		if len(pending) == 0 {
			return nil
		}
		n := rand.Intn(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case s := <-pm.txsyncCh:
			pending[s.p.ID()] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			// Stop tracking peers that cause send failures.
			if err != nil {
				pack.p.Log().Debug("Transaction send failed", "err", err)
				delete(pending, pack.p.ID())
			}
			// Schedule the next send.
			if s := pick(); s != nil {
				send(s)
			}
		case <-pm.quitSync:
			return
		}
	}
}

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
// syncer负责定期与网络同步，下载散列和块，并处理公告处理程序。
func (pm *ProtocolManager) syncer(appId string) {
	//log.Info("---打开区块同步循环", "AppId", appId)
	// Start and ensure cleanup of sync mechanisms
	if appId == "" {
		pm.fetcher.Start()
		defer pm.fetcher.Stop()
		defer pm.downloader.Terminate()
	} else {
		defer pm.SideDownloader[appId].Terminate()
	}

	// Wait for different events to fire synchronisation operations
	forceSync := time.NewTicker(forceSyncCycle)
	defer forceSync.Stop()

	for {
		select {
		case <-pm.newPeerCh:
			// Make sure we have peers to select from, then sync
			if pm.peers.Len() < minDesiredPeerCount {
				break
			}
			go pm.synchronise(pm.peers.BestPeer(appId), appId)

		case <-forceSync.C:
			// Force a sync even if not enough peers are present
			go pm.synchronise(pm.peers.BestPeer(appId), appId)

		case <-pm.noMorePeers[appId]:
			return
		}
	}
}

// synchronise tries to sync up our local block chain with a remote peer.
// synchronise 尝试将本地块链与远程对等点同步。
func (pm *ProtocolManager) synchronise(peer *peer, appId string) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}
	chain, err := pm.ExistsAppId(appId)
	if chain == nil || err != nil {
		return
	}
	download := pm.downloader
	if appId != "" {
		download = pm.SideDownloader[appId]
	}
	// Make sure the peer's TD is higher than our own
	// 确保将要同步的peer的块的工作量比自己的大
	currentBlock := chain.CurrentBlock()
	td := chain.GetTd(currentBlock.Hash(), currentBlock.NumberU64())

	pHead, pTd := peer.Head(appId)
	if pTd.Cmp(td) <= 0 {
		return
	}
	// Otherwise try to sync with the downloader
	mode := downloader.FullSync
	fs := pm.fastSync[appId]
	if atomic.LoadUint32(&fs) == 1 {
		// Fast sync was explicitly requested, and explicitly granted
		mode = downloader.FastSync
	} else if currentBlock.NumberU64() == 0 && chain.CurrentFastBlock().NumberU64() > 0 {
		// The database seems empty as the current block is the genesis. Yet the fast
		// block is ahead, so fast sync was enabled for this node at a certain point.
		// The only scenario where this can happen is if the user manually (or via a
		// bad block) rolled back a fast sync node below the sync point. In this case
		// however it's safe to reenable fast sync.
		atomic.StoreUint32(&fs, 1)
		mode = downloader.FastSync
	}

	if mode == downloader.FastSync {
		// Make sure the peer's total difficulty we are synchronizing is higher.
		if chain.GetTdByHash(chain.CurrentFastBlock().Hash()).Cmp(pTd) >= 0 {
			return
		}
	}

	// Run the sync cycle, and disable fast sync if we've went past the pivot block
	if err := download.Synchronise(peer.id, pHead, pTd, mode); err != nil {
		return
	}
	if atomic.LoadUint32(&fs) == 1 {
		log.Info("Fast sync complete, auto disabling")
		atomic.StoreUint32(&fs, 0)
	}
	atomic.StoreUint32(&pm.acceptTxs, 1) // Mark initial sync done
	if head := chain.CurrentBlock(); head.NumberU64() > 0 {
		// We've completed a sync cycle, notify all peers of new state. This path is
		// essential in star-topology networks where a gateway node needs to notify
		// all its out-of-date peers of the availability of a new block. This failure
		// scenario will most often crop up in private and hackathon networks with
		// degenerate connectivity, but it should be healthy for the mainnet too to
		// more reliably update peers or the local TD state.
		go pm.BroadcastBlock(head, false)
	}
}
