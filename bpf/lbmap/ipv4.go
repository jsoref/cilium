//
// Copyright 2016 Authors of Cilium
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package lbmap

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"unsafe"

	"github.com/cilium/cilium/common"
	"github.com/cilium/cilium/common/bpf"
	"github.com/cilium/cilium/common/types"
)

var (
	Service4Map = bpf.NewMap(common.BPFCiliumMaps+"/cilium_lb4_services",
		bpf.MapTypeHash,
		int(unsafe.Sizeof(Service4Key{})),
		int(unsafe.Sizeof(Service4Value{})),
		maxEntries)
	RevNat4Map = bpf.NewMap(common.BPFCiliumMaps+"/cilium_lb4_reverse_nat",
		bpf.MapTypeHash,
		int(unsafe.Sizeof(RevNat4Key(0))),
		int(unsafe.Sizeof(RevNat4Value{})),
		maxEntries)
)

// Must match 'struct lb4_key' in "bpf/lib/common.h"
type Service4Key struct {
	Address types.IPv4
	Port    uint16
	Slave   uint16
}

func (k Service4Key) IsIPv6() bool           { return false }
func (k Service4Key) Map() *bpf.Map          { return Service4Map }
func (k Service4Key) NewValue() bpf.MapValue { return &Service4Value{} }

func (k Service4Key) GetKeyPtr() unsafe.Pointer {
	return unsafe.Pointer(&k)
}

func (k Service4Key) MapDelete() error {
	return k.Map().Delete(k)
}

func NewService4Key(ip net.IP, port uint16, slave uint16) *Service4Key {
	key := Service4Key{
		Port:  common.Swab16(port),
		Slave: slave,
	}

	copy(key.Address[:], ip.To4())

	return &key
}

// Must match 'struct lb4_service' in "bpf/lib/common.h"
type Service4Value struct {
	Address types.IPv4
	Port    uint16
	Count   uint16
	RevNAT  uint16
}

func NewService4Value(count uint16, target net.IP, port uint16, revNat uint16) *Service4Value {
	svc := Service4Value{
		Count:  count,
		RevNAT: common.Swab16(revNat),
		Port:   common.Swab16(port),
	}

	copy(svc.Address[:], target.To4())

	return &svc
}

func (s Service4Value) GetValuePtr() unsafe.Pointer {
	return unsafe.Pointer(&s)
}

func Service4DumpParser(key []byte, value []byte) (bpf.MapKey, bpf.MapValue, error) {
	keyBuf := bytes.NewBuffer(key)
	valueBuf := bytes.NewBuffer(value)
	svcKey := Service4Key{}
	svcVal := Service4Value{}

	if err := binary.Read(keyBuf, binary.LittleEndian, &svcKey); err != nil {
		return nil, nil, fmt.Errorf("Unable to convert key: %s\n", err)
	}

	svcKey.Port = common.Swab16(svcKey.Port)

	if err := binary.Read(valueBuf, binary.LittleEndian, &svcVal); err != nil {
		return nil, nil, fmt.Errorf("Unable to convert key: %s\n", err)
	}

	svcVal.Port = common.Swab16(svcVal.Port)
	svcVal.RevNAT = common.Swab16(svcVal.RevNAT)

	return &svcKey, &svcVal, nil
}

type RevNat4Key uint16

func NewRevNat4Key(value uint16) RevNat4Key {
	return RevNat4Key(common.Swab16(value))
}

func (k RevNat4Key) IsIPv6() bool           { return false }
func (k RevNat4Key) Map() *bpf.Map          { return RevNat4Map }
func (k RevNat4Key) NewValue() bpf.MapValue { return &RevNat4Value{} }
func (k RevNat4Key) GetKeyPtr() unsafe.Pointer {
	return unsafe.Pointer(&k)
}

type RevNat4Value struct {
	Address types.IPv4
	Port    uint16
}

func (k RevNat4Value) GetValuePtr() unsafe.Pointer {
	return unsafe.Pointer(&k)
}

func NewRevNat4Value(ip net.IP, port uint16) *RevNat4Value {
	revNat := RevNat4Value{
		Port: common.Swab16(port),
	}

	copy(revNat.Address[:], ip.To4())

	return &revNat
}

func RevNat4DumpParser(key []byte, value []byte) (bpf.MapKey, bpf.MapValue, error) {
	var revNat RevNat4Value
	var ukey uint16

	keyBuf := bytes.NewBuffer(key)
	valueBuf := bytes.NewBuffer(value)

	if err := binary.Read(keyBuf, binary.LittleEndian, &ukey); err != nil {
		return nil, nil, fmt.Errorf("Unable to convert key: %s\n", err)
	}
	revKey := NewRevNat4Key(ukey)

	if err := binary.Read(valueBuf, binary.LittleEndian, &revNat); err != nil {
		return nil, nil, fmt.Errorf("Unable to convert value: %s\n", err)
	}

	revNat.Port = common.Swab16(revNat.Port)

	return &revKey, &revNat, nil
}
