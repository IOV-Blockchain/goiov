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

// Package fetcher contains the block announcement based synchronisation.
package fetcher

import (
	"errors"
	"math/rand"
	"time"

	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/consensus"
	"github.com/CarLiveChainCo/goiov/core"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/log"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

const (
	arriveTimeout = 500 * time.Millisecond // Time allowance before an announced block is explicitly requested	// 在显式请求一个声明块之前的时间余量
	gatherSlack   = 100 * time.Millisecond // Interval used to collate almost-expired announces with fetches	// 间隔，用于将快过期的通知与读取进行比较
	fetchTimeout  = 5 * time.Second        // Maximum allotted time to return an explicitly requested block		// 返回显式请求块的最大分配时间
	maxUncleDist  = 7                      // Maximum allowed backward distance from the chain head				// 从链条头向后的最大允许距离
	maxQueueDist  = 32                     // Maximum allowed distance from the chain head to queue				// 链头到队列的最大允许距离
	hashLimit     = 256                    // Maximum number of unique blocks a peer may have announced			// 对等点可能已声明的惟一块的最大数量
	blockLimit    = 64                     // Maximum number of unique blocks a peer may have delivered			// 对等点可能已交付的唯一块的最大数量
)

var (
	errTerminated = errors.New("terminated")
)

// blockRetrievalFn is a callback type for retrieving a block from the local chain.
// 用于从本地链中检索块。
type blockRetrievalFn func(hash common.Hash, appId string) *types.Block

// headerRequesterFn is a callback type for sending a header retrieval request.
// 用于发送报头检索请求。
type headerRequesterFn func(common.Hash, string) error

// bodyRequesterFn is a callback type for sending a body retrieval request.
// 用于发送body检索请求。
type bodyRequesterFn func([]common.Hash, string) error

// headerVerifierFn is a callback type to verify a block's header for fast propagation.
// 用于验证块的标头以进行快速传播。
type headerVerifierFn func(header *types.Header) error

// blockBroadcasterFn is a callback type for broadcasting a block to connected peers.
// 用于向连接的对等体广播块。
type blockBroadcasterFn func(block *types.Block, propagate bool)

// chainHeightFn is a callback type to retrieve the current chain height.
// 用于检索当前的链高。
type chainHeightFn func(appId string) int64

// chainInsertFn is a callback type to insert a batch of blocks into the local chain.
// 用于将一批块插入本地链。
type chainInsertFn func(types.Blocks) (int, error)

// peerDropFn is a callback type for dropping a peer detected as malicious.
// 用于删除被检测为恶意的对等体。
type peerDropFn func(id string)

// announce is the hash notification of the availability of a new block in the
// network.
// announce是网络中新块可用性的哈希通知。
type announce struct {
	hash   common.Hash   // Hash of the block being announced 								 // 正在公布的区块的哈希
	number uint64        // Number of the block being announced (0 = unknown | old protocol) // 公布的块数（0 =未知|旧协议）
	header *types.Header // Header of the block partially reassembled (new protocol)		 // 块的标题部分重新组装（新协议）
	time   time.Time     // Timestamp of the announcement									 // 公告的时间戳

	origin string // Identifier of the peer originating the notification							// 发起通知的对等方的标识符

	fetchHeader headerRequesterFn // Fetcher function to retrieve the header of an announced block  //Fetcher函数用于检索已宣布块的块头
	fetchBodies bodyRequesterFn   // Fetcher function to retrieve the body of an announced block 	//Fetcher函数用于检索已宣布块的主体
	appId       string
}

// headerFilterTask represents a batch of headers needing fetcher filtering.
// headerFilterTask表示需要获取器过滤的一批标头。
type headerFilterTask struct {
	peer    string          // The source peer of block headers		// 块头的源对等体
	headers []*types.Header // Collection of headers to filter		// 要过滤的标头集合
	time    time.Time       // Arrival time of the headers			// 标题的到达时间
}

// headerFilterTask represents a batch of block bodies (transactions and uncles)
// needing fetcher filtering.
// headerFilterTask表示需要获取器过滤的一批块体（事务和叔叔)
type bodyFilterTask struct {
	peer         string                 // The source peer of block bodies
	transactions [][]*types.Transaction // Collection of transactions per block bodies
	uncles       [][]*types.Header      // Collection of uncles per block bodies
	time         time.Time              // Arrival time of the blocks' contents
}

// inject represents a schedules import operation.
// inject表示计划导入操作
type inject struct {
	origin string
	block  *types.Block
}

// Fetcher is responsible for accumulating block announcements from various peers
// and scheduling them for retrieval.
// Fetcher负责从各个对等方累积块通知并安排它们进行检索。
type Fetcher struct {
	// Various event channels
	// 各种活动频道
	notify chan *announce
	inject chan *inject

	blockFilter  chan chan []*types.Block
	headerFilter chan chan *headerFilterTask
	bodyFilter   chan chan *bodyFilterTask

	done chan common.Hash
	quit chan struct{}

	// Announce states
	announces  map[string]int              // Per peer announce counts to prevent memory exhaustion		// 每个对等方宣布计数以防止内存耗尽
	announced  map[common.Hash][]*announce // Announced blocks, scheduled for fetching					// 宣布块，计划提取
	fetching   map[common.Hash]*announce   // Announced blocks, currently fetching						// 宣布块，当前正在获取
	fetched    map[common.Hash][]*announce // Blocks with headers fetched, scheduled for body retrieval // 提取标题的块，计划用于正文检索
	completing map[common.Hash]*announce   // Blocks with headers, currently body-completing 			// 带有标题的块，当前正在完成

	// Block cache
	queue  *prque.Prque            // Queue containing the import operations (block number sorted) 		// 包含导入操作的队列（已排序的块编号）
	queues map[string]int          // Per peer block counts to prevent memory exhaustion 				// 每个对等块计数以防止内存耗尽
	queued map[common.Hash]*inject // Set of already queued blocks (to dedupe imports) 					// 已排队的一组块（重复数据删除导入）

	// Callbacks
	getBlock       blockRetrievalFn   // Retrieves a block from the local chain
	verifyHeader   headerVerifierFn   // Checks if a block's headers have a valid proof of work
	broadcastBlock blockBroadcasterFn // Broadcasts a block to connected peers
	chainHeight    chainHeightFn      // Retrieves the current chain's height
	insertChain    chainInsertFn      // Injects a batch of blocks into the chain
	dropPeer       peerDropFn         // Drops a peer for misbehaving

	// Testing hooks
	announceChangeHook func(common.Hash, bool) // Method to call upon adding or deleting a hash from the announce list
	queueChangeHook    func(common.Hash, bool) // Method to call upon adding or deleting a block from the import queue
	fetchingHook       func([]common.Hash)     // Method to call upon starting a block (eth/61) or header (eth/62) fetch
	completingHook     func([]common.Hash)     // Method to call upon starting a block body fetch (eth/62)
	importedHook       func(*types.Block)      // Method to call upon successful block import (both eth/61 and eth/62)
	blockChain         *core.BlockChain
}

// New creates a block fetcher to retrieve blocks based on hash announcements.
func New(getBlock blockRetrievalFn, verifyHeader headerVerifierFn, broadcastBlock blockBroadcasterFn, chainHeight chainHeightFn, insertChain chainInsertFn, dropPeer peerDropFn, blockChain *core.BlockChain) *Fetcher {
	return &Fetcher{
		notify:         make(chan *announce),
		inject:         make(chan *inject),
		blockFilter:    make(chan chan []*types.Block),
		headerFilter:   make(chan chan *headerFilterTask),
		bodyFilter:     make(chan chan *bodyFilterTask),
		done:           make(chan common.Hash),
		quit:           make(chan struct{}),
		announces:      make(map[string]int),
		announced:      make(map[common.Hash][]*announce),
		fetching:       make(map[common.Hash]*announce),
		fetched:        make(map[common.Hash][]*announce),
		completing:     make(map[common.Hash]*announce),
		queue:          prque.New(),
		queues:         make(map[string]int),
		queued:         make(map[common.Hash]*inject),
		getBlock:       getBlock,
		verifyHeader:   verifyHeader,
		broadcastBlock: broadcastBlock,
		chainHeight:    chainHeight,
		insertChain:    insertChain,
		dropPeer:       dropPeer,
		blockChain:     blockChain,
	}
}

// Start boots up the announcement based synchroniser, accepting and processing
// hash notifications and block fetches until termination requested.
func (f *Fetcher) Start() {
	go f.loop()
}

// Stop terminates the announcement based synchroniser, canceling all pending
// operations.
func (f *Fetcher) Stop() {
	close(f.quit)
}

// Notify announces the fetcher of the potential availability of a new block in
// the network.
func (f *Fetcher) Notify(peer string, appId string, hash common.Hash, number uint64, time time.Time,
	headerFetcher headerRequesterFn, bodyFetcher bodyRequesterFn) error {
	block := &announce{
		hash:        hash,
		number:      number,
		time:        time,
		origin:      peer,
		fetchHeader: headerFetcher,
		fetchBodies: bodyFetcher,
		appId:       appId,
	}
	select {
	case f.notify <- block:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Enqueue tries to fill gaps the the fetcher's future import queue.
func (f *Fetcher) Enqueue(peer string, block *types.Block) error {
	op := &inject{
		origin: peer,
		block:  block,
	}
	select {
	case f.inject <- op:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// FilterHeaders extracts all the headers that were explicitly requested by the fetcher,
// returning those that should be handled differently.
func (f *Fetcher) FilterHeaders(peer string, headers []*types.Header, time time.Time) []*types.Header {
	log.Trace("Filtering headers", "peer", peer, "headers", len(headers))

	// Send the filter channel to the fetcher
	filter := make(chan *headerFilterTask)

	select {
	case f.headerFilter <- filter:
	case <-f.quit:
		return nil
	}
	// Request the filtering of the header list
	select {
	case filter <- &headerFilterTask{peer: peer, headers: headers, time: time}:
	case <-f.quit:
		return nil
	}
	// Retrieve the headers remaining after filtering
	select {
	case task := <-filter:
		return task.headers
	case <-f.quit:
		return nil
	}
}

// FilterBodies extracts all the block bodies that were explicitly requested by
// the fetcher, returning those that should be handled differently.
func (f *Fetcher) FilterBodies(peer string, transactions [][]*types.Transaction, uncles [][]*types.Header, time time.Time) ([][]*types.Transaction, [][]*types.Header) {
	log.Trace("Filtering bodies", "peer", peer, "txs", len(transactions), "uncles", len(uncles))

	// Send the filter channel to the fetcher
	filter := make(chan *bodyFilterTask)

	select {
	case f.bodyFilter <- filter:
	case <-f.quit:
		return nil, nil
	}
	// Request the filtering of the body list
	select {
	case filter <- &bodyFilterTask{peer: peer, transactions: transactions, uncles: uncles, time: time}:
	case <-f.quit:
		return nil, nil
	}
	// Retrieve the bodies remaining after filtering
	select {
	case task := <-filter:
		return task.transactions, task.uncles
	case <-f.quit:
		return nil, nil
	}
}

// Loop is the main fetcher loop, checking and processing various notification
// events.
// 检查和处理各种通知事件
func (f *Fetcher) loop() {
	// Iterate the block fetching until a quit is requested
	// 迭代获取块，直到请求退出
	fetchTimer := time.NewTimer(0)
	completeTimer := time.NewTimer(0)

	for {
		// Clean up any expired block fetches
		// 清除任何过期的块获取
		for hash, announce := range f.fetching {
			if time.Since(announce.time) > fetchTimeout {
				f.forgetHash(hash)
			}
		}
		// Import any queued blocks that could potentially fit
		// 检查收到的块并上链（是否已在链上是否是过期的块）
		// 导入任何可能适合的队列块
		for !f.queue.Empty() {
			op := f.queue.PopItem().(*inject)
			hash := op.block.Hash()
			fheight := f.chainHeight(op.block.Header().Appid)
			// 没有找到对应的链
			if fheight < 0 {
				f.forgetBlock(hash)
				continue
			}
			if f.queueChangeHook != nil {
				f.queueChangeHook(op.block.Hash(), false)
			}
			height := uint64(fheight)
			// If too high up the chain or phase, continue later
			// 如果链条或阶段太高，稍后继续
			number := op.block.NumberU64()
			if number > height+1 {
				f.queue.Push(op, -float32(op.block.NumberU64()))
				if f.queueChangeHook != nil {
					f.queueChangeHook(op.block.Hash(), true)
				}
				break
			}
			// Otherwise if fresh and still unknown, try and import
			if number+maxUncleDist < height || f.getBlock(hash, op.block.Header().Appid) != nil {
				f.forgetBlock(hash)
				continue
			}
			f.insert(op.origin, op.block)
		}
		// Wait for an outside event to occur
		select {
		case <-f.quit:
			// Fetcher terminating, abort all operations
			return

		case notification := <-f.notify:
			// A block was announced, make sure the peer isn't DOSing us
			propAnnounceInMeter.Mark(1)

			count := f.announces[notification.origin] + 1
			if count > hashLimit {
				log.Debug("Peer exceeded outstanding announces", "peer", notification.origin, "limit", hashLimit)
				propAnnounceDOSMeter.Mark(1)
				break
			}
			// If we have a valid block number, check that it's potentially useful
			if notification.number > 0 {
				height := f.chainHeight(notification.appId)
				if height < 0 {
					break
				}
				if dist := int64(notification.number) - int64(height); dist < -maxUncleDist || dist > maxQueueDist {
					log.Debug("Peer discarded announcement", "peer", notification.origin, "number", notification.number, "hash", notification.hash, "distance", dist)
					propAnnounceDropMeter.Mark(1)
					break
				}
			}
			// All is well, schedule the announce if block's not yet downloading
			if _, ok := f.fetching[notification.hash]; ok {
				break
			}
			if _, ok := f.completing[notification.hash]; ok {
				break
			}
			f.announces[notification.origin] = count
			f.announced[notification.hash] = append(f.announced[notification.hash], notification)
			if f.announceChangeHook != nil && len(f.announced[notification.hash]) == 1 {
				f.announceChangeHook(notification.hash, true)
			}
			if len(f.announced) == 1 {
				f.rescheduleFetch(fetchTimer)
			}

		case op := <-f.inject:
			// A direct block insertion was requested, try and fill any pending gaps
			// 请求直接插入块，尝试填充任何挂起的空白
			propBroadcastInMeter.Mark(1)
			f.enqueue(op.origin, op.block)

		case hash := <-f.done:
			// A pending import finished, remove all traces of the notification
			// 待导入完成，删除通知的所有跟踪
			f.forgetHash(hash)
			f.forgetBlock(hash)

			// 每隔一段时间从announced每个节点中随机选取一个块hash，如果本链上没有，则向其他节点请求该块的头
		case <-fetchTimer.C:
			// At least one block's timer ran out, check for needing retrieval
			request := make(map[string][]*announce)

			for hash, announces := range f.announced {
				if time.Since(announces[0].time) > arriveTimeout-gatherSlack {
					// Pick a random peer to retrieve from, reset all others
					// 随机选择要检索的对等点，重置所有其他对等点
					announce := announces[rand.Intn(len(announces))]
					f.forgetHash(hash)

					// If the block still didn't arrive, queue for fetching
					// 如果块还没有到达，就排队取
					if f.getBlock(hash, announce.appId) == nil {
						request[announce.origin] = append(request[announce.origin], announce)
						f.fetching[hash] = announce
						fetchHeader := f.fetching[hash].fetchHeader
						go func() {
							headerFetchMeter.Mark(1)
							fetchHeader(hash, announce.appId)
						}()
					}
				}
			}
			// Send out all block header requests
			for peer, ann := range request {
				var hashes []common.Hash
				for _, a := range ann {
					hashes = append(hashes, a.hash)
				}
				log.Trace("Fetching scheduled headers", "peer", peer, "list", hashes)

				// Create a closure of the fetch and schedule in on a new thread
				fetchHeader, hashes := f.fetching[ann[0].hash].fetchHeader, hashes
				go func() {
					if f.fetchingHook != nil {
						f.fetchingHook(hashes)
					}
					for _, an := range ann {
						headerFetchMeter.Mark(1)
						fetchHeader(an.hash, an.appId) // Suboptimal, but protocol doesn't allow batch header retrievals
					}
				}()
			}
			// Schedule the next fetch if blocks are still pending
			f.rescheduleFetch(fetchTimer)

		case <-completeTimer.C:
			// At least one header's timer ran out, retrieve everything
			request := make(map[string]map[string][]common.Hash)

			for hash, announces := range f.fetched {
				// Pick a random peer to retrieve from, reset all others
				announce := announces[rand.Intn(len(announces))]
				f.forgetHash(hash)

				// If the block still didn't arrive, queue for completion
				if f.getBlock(hash, announce.appId) == nil {
					if _, ok := request[announce.origin]; !ok {
						request[announce.origin] = make(map[string][]common.Hash)
					}
					request[announce.origin][announce.appId] = append(request[announce.origin][announce.appId], hash)
					f.completing[hash] = announce
				}
			}
			// Send out all block body requests
			for peer, idMap := range request {
				for appId, hashes := range idMap {
					log.Trace("Fetching scheduled bodies", "peer", peer, "list", hashes)

					// Create a closure of the fetch and schedule in on a new thread
					if f.completingHook != nil {
						f.completingHook(hashes)
					}
					bodyFetchMeter.Mark(int64(len(hashes)))
					go f.completing[hashes[0]].fetchBodies(hashes, appId)
				}
			}
			// Schedule the next fetch if blocks are still pending
			f.rescheduleComplete(completeTimer)

		case filter := <-f.headerFilter:
			// Headers arrived from a remote peer. Extract those that were explicitly
			// requested by the fetcher, and return everything else so it's delivered
			// to other parts of the system.
			var task *headerFilterTask
			select {
			case task = <-filter:
			case <-f.quit:
				return
			}
			headerFilterInMeter.Mark(int64(len(task.headers)))

			// Split the batch of headers into unknown ones (to return to the caller),
			// known incomplete ones (requiring body retrievals) and completed blocks.
			unknown, incomplete, complete := []*types.Header{}, []*announce{}, []*types.Block{}
			for _, header := range task.headers {
				hash := header.Hash()

				// Filter fetcher-requested headers from other synchronisation algorithms
				if announce := f.fetching[hash]; announce != nil && announce.origin == task.peer && f.fetched[hash] == nil && f.completing[hash] == nil && f.queued[hash] == nil {
					// If the delivered header does not match the promised number, drop the announcer
					if header.Number.Uint64() != announce.number {
						log.Trace("Invalid block number fetched", "peer", announce.origin, "hash", header.Hash(), "announced", announce.number, "provided", header.Number)
						// joker
						log.Info("-----fetcher---dropPeer", "headerNumber", task.headers[0].Number)
						f.dropPeer(announce.origin)
						f.forgetHash(hash)
						continue
					}
					// Only keep if not imported by other means
					if f.getBlock(hash, header.Appid) == nil {
						announce.header = header
						announce.time = task.time

						// If the block is empty (header only), short circuit into the final import queue
						// 如果该块是空的(只有头部)，则短路到最终导入队列
						if header.TxHash == types.DeriveSha(types.Transactions{}) && header.UncleHash == types.CalcUncleHash([]*types.Header{}) {
							log.Trace("Block empty, skipping body retrieval", "peer", announce.origin, "number", header.Number, "hash", header.Hash())

							block := types.NewBlockWithHeader(header)
							block.ReceivedAt = task.time

							complete = append(complete, block)
							f.completing[hash] = announce
							continue
						}
						// Otherwise add to the list of blocks needing completion
						incomplete = append(incomplete, announce)
					} else {
						log.Trace("Block already imported, discarding header", "peer", announce.origin, "number", header.Number, "hash", header.Hash())
						f.forgetHash(hash)
					}
				} else {
					// Fetcher doesn't know about it, add to the return list
					unknown = append(unknown, header)
				}
			}
			headerFilterOutMeter.Mark(int64(len(unknown)))
			select {
			case filter <- &headerFilterTask{headers: unknown, time: task.time}:
			case <-f.quit:
				return
			}
			// Schedule the retrieved headers for body completion
			// 为body的拼装安排header
			for _, announce := range incomplete {
				hash := announce.header.Hash()
				if _, ok := f.completing[hash]; ok {
					continue
				}
				f.fetched[hash] = append(f.fetched[hash], announce)
				if len(f.fetched) == 1 {
					f.rescheduleComplete(completeTimer)
				}
			}
			// Schedule the header-only blocks for import
			// 为导入设置只包含header的块
			for _, block := range complete {
				if announce := f.completing[block.Hash()]; announce != nil {
					f.enqueue(announce.origin, block)
				}
			}

		case filter := <-f.bodyFilter:
			// Block bodies arrived, extract any explicitly requested blocks, return the rest
			var task *bodyFilterTask
			select {
			case task = <-filter:
			case <-f.quit:
				return
			}
			bodyFilterInMeter.Mark(int64(len(task.transactions)))

			blocks := []*types.Block{}
			for i := 0; i < len(task.transactions) && i < len(task.uncles); i++ {
				// Match up a body to any possible completion request
				matched := false

				for hash, announce := range f.completing {
					if f.queued[hash] == nil {
						txnHash := types.DeriveSha(types.Transactions(task.transactions[i]))
						uncleHash := types.CalcUncleHash(task.uncles[i])

						if txnHash == announce.header.TxHash && uncleHash == announce.header.UncleHash && announce.origin == task.peer {
							// Mark the body matched, reassemble if still unknown
							matched = true

							if f.getBlock(hash, announce.appId) == nil {
								block := types.NewBlockWithHeader(announce.header).WithBody(task.transactions[i], task.uncles[i])
								block.ReceivedAt = task.time

								blocks = append(blocks, block)
							} else {
								f.forgetHash(hash)
							}
						}
					}
				}
				if matched {
					task.transactions = append(task.transactions[:i], task.transactions[i+1:]...)
					task.uncles = append(task.uncles[:i], task.uncles[i+1:]...)
					i--
					continue
				}
			}

			bodyFilterOutMeter.Mark(int64(len(task.transactions)))
			select {
			case filter <- task:
			case <-f.quit:
				return
			}
			// Schedule the retrieved blocks for ordered import
			for _, block := range blocks {
				if announce := f.completing[block.Hash()]; announce != nil {
					f.enqueue(announce.origin, block)
				}
			}
		}
	}
}

// rescheduleFetch resets the specified fetch timer to the next announce timeout.
// rescheduleFetch将指定的获取计时器重置为下一次通知超时。
func (f *Fetcher) rescheduleFetch(fetch *time.Timer) {
	// Short circuit if no blocks are announced
	if len(f.announced) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.announced {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	fetch.Reset(arriveTimeout - time.Since(earliest))
}

// rescheduleComplete resets the specified completion timer to the next fetch timeout.
// 将指定的完成计时器重置为下一次获取超时。
func (f *Fetcher) rescheduleComplete(complete *time.Timer) {
	// Short circuit if no headers are fetched
	if len(f.fetched) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.fetched {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	complete.Reset(gatherSlack - time.Since(earliest))
}

// enqueue schedules a new future import operation, if the block to be imported
// has not yet been seen.
// 如果要导入的块尚未被看到，enqueue将调度一个新的未来导入操作。
func (f *Fetcher) enqueue(peer string, block *types.Block) {
	hash := block.Hash()

	// Ensure the peer isn't DOSing us
	count := f.queues[peer] + 1
	if count > blockLimit {
		log.Debug("Discarded propagated block, exceeded allowance", "peer", peer, "number", block.Number(), "hash", hash, "limit", blockLimit)
		propBroadcastDOSMeter.Mark(1)
		f.forgetHash(hash)
		return
	}
	height := f.chainHeight(block.Header().Appid)
	if height < 0 {
		return
	}
	// Discard any past or too distant blocks
	if dist := int64(block.NumberU64()) - int64(height); dist < -maxUncleDist || dist > maxQueueDist {
		log.Debug("Discarded propagated block, too far away", "peer", peer, "number", block.Number(), "hash", hash, "distance", dist)
		propBroadcastDropMeter.Mark(1)
		f.forgetHash(hash)
		return
	}
	// Schedule the block for future importing
	// 为将来的导入安排块
	if _, ok := f.queued[hash]; !ok {
		op := &inject{
			origin: peer,
			block:  block,
		}
		f.queues[peer] = count
		f.queued[hash] = op
		f.queue.Push(op, -float32(block.NumberU64()))
		if f.queueChangeHook != nil {
			f.queueChangeHook(op.block.Hash(), true)
		}
		log.Debug("Queued propagated block", "peer", peer, "number", block.Number(), "hash", hash, "queued", f.queue.Size())
	}
}

// insert spawns a new goroutine to run a block insertion into the chain. If the
// block's number is at the same height as the current import phase, it updates
// the phase states accordingly.
func (f *Fetcher) insert(peer string, block *types.Block) {
	hash := block.Hash()

	// Run the import on a new thread
	log.Debug("Importing propagated block", "peer", peer, "number", block.Number(), "hash", hash)
	go func() {
		defer func() { f.done <- hash }()

		// If the parent's unknown, abort insertion
		parent := f.getBlock(block.ParentHash(), block.Header().Appid)
		if parent == nil {
			log.Debug("Unknown parent of propagated block", "peer", peer, "number", block.Number(), "hash", hash, "parent", block.ParentHash())
			return
		}
		// Quickly validate the header and propagate the block if it passes
		switch err := f.verifyHeader(block.Header()); err {
		case nil:
			// All ok, quickly propagate to our peers
			propBroadcastOutTimer.UpdateSince(block.ReceivedAt)
			go f.broadcastBlock(block, true)

		case consensus.ErrFutureBlock:
			// Weird future block, don't fail, but neither propagate

		default:
			// Something went very wrong, drop the peer
			log.Debug("Propagated block verification failed", "peer", peer, "number", block.Number(), "hash", hash, "err", err)
			f.dropPeer(peer)
			return
		}
		// Run the actual import and log any issues
		if _, err := f.insertChain(types.Blocks{block}); err != nil {
			log.Debug("Propagated block import failed", "peer", peer, "number", block.Number(), "hash", hash, "err", err)
			return
		}
		// If import succeeded, broadcast the block
		propAnnounceOutTimer.UpdateSince(block.ReceivedAt)
		go f.broadcastBlock(block, false)

		// Invoke the testing hook if needed
		if f.importedHook != nil {
			f.importedHook(block)
		}
	}()
}

// forgetHash removes all traces of a block announcement from the fetcher's
// internal state.
func (f *Fetcher) forgetHash(hash common.Hash) {
	// Remove all pending announces and decrement DOS counters
	for _, announce := range f.announced[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.announced, hash)
	if f.announceChangeHook != nil {
		f.announceChangeHook(hash, false)
	}
	// Remove any pending fetches and decrement the DOS counters
	if announce := f.fetching[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.fetching, hash)
	}

	// Remove any pending completion requests and decrement the DOS counters
	for _, announce := range f.fetched[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.fetched, hash)

	// Remove any pending completions and decrement the DOS counters
	if announce := f.completing[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.completing, hash)
	}
}

// forgetBlock removes all traces of a queued block from the fetcher's internal
// state.
func (f *Fetcher) forgetBlock(hash common.Hash) {
	if insert := f.queued[hash]; insert != nil {
		f.queues[insert.origin]--
		if f.queues[insert.origin] == 0 {
			delete(f.queues, insert.origin)
		}
		delete(f.queued, hash)
	}
}
