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

package appmanager

import (
	"github.com/CarLiveChainCo/goiov/common"
	"github.com/CarLiveChainCo/goiov/crypto"
	"testing"
)

var (
	key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	name     = "my name on AppManager"
	addr     = crypto.PubkeyToAddress(key.PublicKey)
	testAddr = common.HexToAddress("0x1234123412341234123412341234123412341234")
)

func TestAppManager(t *testing.T) {
	//contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}}, 10000000)
	//
	//appManager, err := NewAppManager(MainNetAppManagerAddress, contractBackend)
	//if err != nil {
	//	t.Fatalf("can't deploy root registry: %v", err)
	//}
	//contractBackend.Commit()

	//// Set name.
	//if _, err := appManager.Name(name); err != nil {
	//	t.Fatalf("can't setName: %v", err)
	//}
	//contractBackend.Commit()
	//
	//// Try to resolve the name to an address
	//getName, err := appManager.getName()
	//if err != nil {
	//	t.Fatalf("getName no error, got %v", err)
	//}
	//if getName != name {
	//	t.Fatalf("getName doesn't equal name")
	//}

}
