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

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	// Ethereum Foundation Go Bootnodes
	"enode://7fdb80ec20f78d1d6d5783c78a27474142e3a47f6ea64daab34f95b95c4abf4c5ebc0c9be2fab2654a72dd57387d9b1c5b7f37982e037142608f074e65fe519d@47.52.241.137:30303",
	"enode://68b9cafa324dd22c3f491bf34b262f47cdd4eef9b518024ffecf59462006c0ab60e5e5c770b128e43107ed845f7f31718be9cf594e79199023ec7d1dcebd372b@47.52.70.84:30303",
	"enode://e19aa9651a6db589c81c29db50c6f24de8375ff170bf14abdbcd7952c2814d948beff4746d1b3e27c262708b5432d0288ca549073e94bc091cdf4e07dbdc2c12@120.77.139.250:30303",
	"enode://7c65a6d77a4385499fd757104f62b343651786ad1b5aa7f6a04a2a475a0d98e6a98434bc20a39a210db93849057235990705a14a32bc3847bd6f393283198f07@47.245.30.101:30303",
}


/*
"enode://4c29256f6590975f27977e98b9f56beab285b089a9cbaa851083de97a4cfb8efce12ab94b7a7c0ffce364399eedec3dfb00375094a1a3f2941ab5dc842ca6ce3@47.52.70.84:30000",
	"enode://03f033f7fdb511ec43f4d8a13866b2087271747d80b046e8ba4dbdba06e5d25199e20d5081f328cd135c56dfa6d5578b6db82bd762641f473bab7ea0be6c808c@120.77.139.250:30000",
	"enode://9efed95869387c81e504796f135ef225ac19b71bb8e08bb80cffc9880b66e631ebdc3744fbda65a9e4012ca4a961ad4a5a9f244b4056311c861285bbcc489879@47.245.30.101:30000",
	"enode://82597bacbdcded5bc120cf31cdc2dfec4181084376d6449b097a71472e9b17c1a560950cd3da54904d7d5b438c49344ce620a7fb9af53bd87145e1df10283377@47.52.241.137:30000",
	"enode://38e2975e898e0e66a0e5426afae81a66adbd0c6aec20120766b646425040c8b633acf28668c8a8bc47abc0d6609396e57c6cdf4491122492a7ee8f9c575a413d@47.52.70.84:30303",
	"enode://db9eb0709df2f493d42a94d825dd865aabcb7c123b5c981c9cd6a84935855ffd8e1677fec7f285b786599cb72e3c5285629b5c60d1bea8829fd5aab78fbdfa7d@120.77.139.250:30303",
	"enode://4223300ed51784d108132b5a7c9abdfb31d42baa0e2fa2e781216febfcacb49759c18aa79111c2d9b6642915cbbf8f7adf65445b33f0298da69aaa86c5e40fd3@47.245.30.101:30303",
*/

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// IOV test network.
var TestnetBootnodes = []string{
	"enode://6bf3d26e4afcdf5373278bf1de89f1e8a6d16d0d80004cab38fa53af5bf4d4bcb717a383586e9451186464a1e01f7dca3d053ce4f9cb5c750779eec3311d9605@47.107.245.170:30000",
	"enode://1f246f5d3442592345bc0fc81f3c6818c6dd6f4f7989c6a49a45b37d95ac0f93fea70063e7d6732d1818641a850fac0f13f563a93cd61c268f4de8fcd53f608a@47.112.25.169:30000",
	"enode://0838e1c3b718edd2eb05290e3ad8eb27e70aee308e9334d07a885739451a59c8eee6e684ce318b7eb1ee18502c58f28d0727180b6231a9562c1ae72b63a1f9b3@47.112.25.169:30303",
	"enode://62774a32f314cfa9edefc9879c1ae696d045d67fbcc2c1cb8f0c42ca5dae7f76f85afef695a14415387ce984a0fe5cf6e3360c32a6e88551611e8e5d1c4628a9@47.107.245.170:30303",
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}
