package cc

import (
	"fmt"
	"hash/maphash"
	"strings"
	"sync"
)

const (
	MaxPartitions     = 10000
	DefaultPartitions = 100
)

func GenPartitions(partitions uint64) uint64 {
	if partitions == 0 {
		partitions = DefaultPartitions
	}
	if partitions > MaxPartitions {
		partitions = MaxPartitions
	}
	return partitions
}

type Partition interface {
	Keys() []any
}

type TxPartitionManager struct {
	partitions uint64
	locks      []sync.Mutex
}

func NewTxPartitionManager(partitions uint64) *TxPartitionManager {
	partitions = GenPartitions(partitions)
	return &TxPartitionManager{
		partitions: partitions,
		locks:      make([]sync.Mutex, partitions),
	}
}

func (mgr *TxPartitionManager) Partition(keys ...any) uint64 {
	var h maphash.Hash
	var sb strings.Builder
	for _, key := range keys {
		sb.WriteString(fmt.Sprintf("%v", key))
	}
	_, _ = h.WriteString(sb.String())
	partition := h.Sum64() % mgr.partitions
	return partition
}

func (mgr *TxPartitionManager) Lock(partition uint64) {
	mgr.locks[partition].Lock()
}

func (mgr *TxPartitionManager) Unlock(partition uint64) {
	mgr.locks[partition].Unlock()
}
