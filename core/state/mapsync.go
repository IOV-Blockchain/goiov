package state

import (
	"sync"
)

type MapsyncInfo struct {
	Key   interface{}
	Value interface{}
}

type Mapsync struct {
	mapValue map[interface{}]interface{}

	lock sync.RWMutex
}

func NewMapsync() Mapsync {
	m := Mapsync{}
	m.mapValue = make(map[interface{}]interface{})
	return m

}

func (this *Mapsync) Get(key interface{}) interface{} {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.mapValue[key]
}

func (this *Mapsync) Set(key interface{}, value interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.mapValue[key] = value
}
func (this *Mapsync) Delete(key interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()
	delete(this.mapValue, key)
}

func (this *Mapsync) GetRangeObj() []MapsyncInfo {
	this.lock.Lock()
	defer this.lock.Unlock()
	arrinfo := make([]MapsyncInfo, 0)
	for k, v := range this.mapValue {
		arrinfo = append(arrinfo, MapsyncInfo{Key: k, Value: v})
	}
	return arrinfo
}
