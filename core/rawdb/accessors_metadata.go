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
// 映像，链配置数据，链版本
package rawdb

import (
	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/log"
	"github.com/CarLiveChainCo/goiov/params"
	"github.com/CarLiveChainCo/goiov/rlp"
	"encoding/json"
)

// ReadDatabaseVersion retrieves the version number of the database.
func ReadDatabaseVersion(db DatabaseReader) int {
	var version int

	enc, _ := db.Get(databaseVerisionKey)
	rlp.DecodeBytes(enc, &version)

	return version
}

// WriteDatabaseVersion stores the version number of the database
func WriteDatabaseVersion(db DatabaseWriter, version int) {
	enc, _ := rlp.EncodeToBytes(version)
	if err := db.Put(databaseVerisionKey, enc); err != nil {
		log.Crit("Failed to store the database version", "err", err)
	}
}

// ReadChainConfig retrieves the consensus settings based on the given genesis hash.
//func ReadChainConfig(db DatabaseReader, hash common.Hash, appid ...string) *params.ChainConfig {
//	var key = append(configPrefix, hash[:]...)
//	if len(appid) == 1 && appid[0] != "" {
//		key = append([]byte(appid[0]), key...)
//	}
//	log.Info("---读链config数据", "appid", appid[0])
//	data, _ := db.Get(key)
//	if len(data) == 0 {
//		return nil
//	}
//	var config params.ChainConfig
//	if err := json.Unmarshal(data, &config); err != nil {
//		log.Error("Invalid chain config JSON", "hash", hash, "err", err)
//		return nil
//	}
//	return &config
//}
func ReadChainConfig(db DatabaseReader, hash common.Hash, appid ...string) *params.ChainConfig {
	key := ""
	if len(appid) == 1 {
		key = appid[0]
	}
	if cMap := ReadAllChainConfig(db); cMap != nil {
		if config, ok := cMap[key]; ok {
			return config
		}
	}
	return nil
}

func ReadAllChainConfig(db DatabaseReader) map[string]*params.ChainConfig {
	var key = configPrefix
	var cMap = make(map[string]*params.ChainConfig)
	data, _ := db.Get(key)
	if len(data) == 0 {
		return cMap
	}
	if err := json.Unmarshal(data, &cMap); err != nil {
		log.Error("Invalid chain config JSON", "err", err)
		return nil
	}
	return cMap
}

// WriteChainConfig writes the chain config settings to the database.
//func WriteChainConfig(db DatabaseWriter, hash common.Hash, cfg *params.ChainConfig, appid ...string) {
//	if cfg == nil {
//		return
//	}
//	data, err := json.Marshal(cfg)
//	if err != nil {
//		log.Crit("Failed to JSON encode chain config", "err", err)
//	}
//	var key = append(configPrefix, hash[:]...)
//	if len(appid) == 1 && appid[0] != "" {
//		key = append([]byte(appid[0]), key...)
//	}
//	log.Info("---写链config数据", "appid", appid[0])
//	if err := db.Put(key, data); err != nil {
//		log.Crit("Failed to store chain config", "err", err)
//	}
//}
func WriteChainConfig(db DatabaseWriter, hash common.Hash, cfg map[string]*params.ChainConfig, appid ...string) {
	if cfg == nil {
		return
	}
	var key = configPrefix
	enc, err := json.Marshal(cfg)
	if err != nil {
		log.Crit("Failed to JSON encode chain config", "err", err)
	}
	if err := db.Put(key, enc); err != nil {
		log.Crit("Failed to store chain config", "err", err)
	}
	log.Info("---写链config数据", "appId", appid[0])
}

// ReadPreimage retrieves a single preimage of the provided hash.
func ReadPreimage(db DatabaseReader, hash common.Hash, appid ...string) []byte {
	var key = append(preimagePrefix, hash.Bytes()...)
	if len(appid) == 1 && appid[0] != "" {
		key = append([]byte(appid[0]), key...)
	}
	data, _ := db.Get(key)
	return data
}

// WritePreimages writes the provided set of preimages to the database. `number` is the
// current block number, and is used for debug messages only.
func WritePreimages(db DatabaseWriter, number uint64, preimages map[common.Hash][]byte, appid ...string) {
	for hash, preimage := range preimages {
		var key = append(preimagePrefix, hash.Bytes()...)
		if len(appid) == 1 && appid[0] != "" {
			key = append([]byte(appid[0]), key...)
		}
		if err := db.Put(key, preimage); err != nil {
			log.Crit("Failed to store trie preimage", "err", err)
		}
	}
	preimageCounter.Inc(int64(len(preimages)))
	preimageHitCounter.Inc(int64(len(preimages)))
}

func WriteAppId(db DatabaseWriter, appId []string) {
	enc, _ := rlp.EncodeToBytes(appId)
	if err := db.Put(appIdKey, enc); err != nil {
		log.Crit("Failed to store the AppId", "err", err)
	}
}

func ReadAppId(db DatabaseReader) []string {
	var appId []string
	enc, _ := db.Get(appIdKey)
	rlp.DecodeBytes(enc, &appId)
	return appId
}
