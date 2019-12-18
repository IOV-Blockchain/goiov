// Copyright 2018 The go-ethereum Authors
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

package rawdb

import (
	"encoding/binary"

	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/core/types"
	"github.com/CarLiveChainCo/goiov/log"
	"github.com/CarLiveChainCo/goiov/rlp"
)

// ReadTxLookupEntry retrieves the positional metadata associated with a transaction
// hash to allow retrieving the transaction or receipt by hash.
func ReadTxLookupEntry(db DatabaseReader, hash common.Hash , appid ...string) (common.Hash, uint64, uint64) {
	var key = append(txLookupPrefix, hash.Bytes()...)
	data, _ := db.Get(key)
	if len(data) == 0 {
		return common.Hash{}, 0, 0
	}
	var entry TxLookupEntry
	if err := rlp.DecodeBytes(data, &entry); err != nil {
		log.Error("Invalid transaction lookup entry RLP", "hash", hash, "err", err)
		return common.Hash{}, 0, 0
	}
	return entry.BlockHash, entry.BlockIndex, entry.Index
}

// WriteTxLookupEntries stores a positional metadata for every transaction from
// a block, enabling hash based transaction and receipt lookups.
func WriteTxLookupEntries(db DatabaseWriter, block *types.Block , appid ...string) {
	for i, tx := range block.Transactions() {
		entry := TxLookupEntry{
			BlockHash:  block.Hash(),
			BlockIndex: block.NumberU64(),
			Index:      uint64(i),
		}
		data, err := rlp.EncodeToBytes(entry)
		if err != nil {
			log.Crit("Failed to encode transaction lookup entry", "err", err)
		}
		var key = append(txLookupPrefix, tx.Hash().Bytes()...)
		if err := db.Put(key , data); err != nil {
			log.Crit("Failed to store transaction lookup entry", "err", err)
		}
	}
}

// DeleteTxLookupEntry removes all transaction data associated with a hash.
func DeleteTxLookupEntry(db DatabaseDeleter, hash common.Hash , appid ...string) {
	var key = append(txLookupPrefix, hash.Bytes()...)
	db.Delete(key)
}

// ReadTransaction retrieves a specific transaction from the database, along with
// its added positional metadata.
func ReadTransaction(db DatabaseReader, hash common.Hash , appid ...string) (*types.Transaction, common.Hash, uint64, uint64) {
	blockHash, blockNumber, txIndex := ReadTxLookupEntry(db, hash)
	if blockHash == (common.Hash{}) {
		return nil, common.Hash{}, 0, 0
	}
	body := ReadBody(db, blockHash, blockNumber)
	if body == nil || len(body.Transactions) <= int(txIndex) {
		log.Error("Transaction referenced missing", "number", blockNumber, "hash", blockHash, "index", txIndex)
		return nil, common.Hash{}, 0, 0
	}
	return body.Transactions[txIndex], blockHash, blockNumber, txIndex
}

// ReadReceipt retrieves a specific transaction receipt from the database, along with
// its added positional metadata.
func ReadReceipt(db DatabaseReader, hash common.Hash , appid ...string) (*types.Receipt, common.Hash, uint64, uint64) {
	var key string
	if len(appid) == 1 && appid[0] != "" {
		key = appid[0]
	}
	blockHash, blockNumber, receiptIndex := ReadTxLookupEntry(db, hash ,key)
	if blockHash == (common.Hash{}) {
		return nil, common.Hash{}, 0, 0
	}
	receipts := ReadReceipts(db, blockHash, blockNumber , key)
	if len(receipts) <= int(receiptIndex) {
		log.Error("Receipt refereced missing", "number", blockNumber, "hash", blockHash, "index", receiptIndex)
		return nil, common.Hash{}, 0, 0
	}
	return receipts[receiptIndex], blockHash, blockNumber, receiptIndex
}

// ReadBloomBits retrieves the compressed bloom bit vector belonging to the given
// section and bit index from the.
func ReadBloomBits(db DatabaseReader, bit uint, section uint64, head common.Hash , appid ...string) ([]byte, error) {
	key := append(append(bloomBitsPrefix, make([]byte, 10)...), head.Bytes()...)
	if len(appid) == 1 && appid[0] != "" {
		key = append([]byte(appid[0]) , key...)
	}
	binary.BigEndian.PutUint16(key[1:], uint16(bit))
	binary.BigEndian.PutUint64(key[3:], section)

	return db.Get(key)
}

// WriteBloomBits stores the compressed bloom bits vector belonging to the given
// section and bit index.
func WriteBloomBits(db DatabaseWriter, bit uint, section uint64, head common.Hash, bits []byte , appid ...string) {
	key := append(append(bloomBitsPrefix, make([]byte, 10)...), head.Bytes()...)
	if len(appid) == 1 && appid[0] != "" {
		key = append([]byte(appid[0]) , key...)
	}
	binary.BigEndian.PutUint16(key[1:], uint16(bit))
	binary.BigEndian.PutUint64(key[3:], section)

	if err := db.Put(key, bits); err != nil {
		log.Crit("Failed to store bloom bits", "err", err)
	}
}
