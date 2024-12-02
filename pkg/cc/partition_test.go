package cc

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxPartitionManager(t *testing.T) {
	partitions := uint64(100)
	prtMgr := NewTxPartitionManager(partitions)

	concurrency := 100000
	r := rand.New(rand.NewSource(42))
	counters := make([]int, partitions)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for range concurrency {
		perm := r.Perm(int(partitions))
		go func() {
			defer wg.Done()
			for _, partition := range perm {
				prtMgr.Lock(uint64(partition))
				counters[partition]++
				prtMgr.Unlock(uint64(partition))
			}
		}()
	}
	wg.Wait()

	for i, count := range counters {
		require.Equal(t, concurrency, count, i)
	}
}
