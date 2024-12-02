package cc

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxOriginManager(t *testing.T) {
	partitions := uint64(10)
	clockMgr := NewTxClockManager(partitions)
	prtMgr := NewTxPartitionManager(partitions)
	originMgr := NewTxOriginManager(partitions, clockMgr, prtMgr)

	serviceA := "service-a"
	serviceB := "service-b"
	serviceC := "service-c"
	services := []string{serviceA, serviceB, serviceC}
	for _, service := range services {
		originMgr.Init(service)
	}

	concurrency := 10000
	expected := []int{}
	for i := range concurrency {
		expected = append(expected, i)
	}
	r := rand.New(rand.NewSource(42))
	for partition := range partitions {
		for _, service := range services {
			result := []int{}
			perm := r.Perm(concurrency)
			var wg sync.WaitGroup
			wg.Add(concurrency * 2)
			for _, ts := range perm {
				go func() {
					defer wg.Done()
					ok := originMgr.Acquire(NewWaitMsg(partition, service, uint64(ts+1)))
					require.True(t, ok)
					result = append(result, ts)
					originMgr.Release(partition, service)
				}()
				// outdated message
				go func() {
					defer wg.Done()
					ok := originMgr.Acquire(NewWaitMsg(partition, service, uint64(0)))
					require.False(t, ok)
					// originMgr.Release(partition, service)
				}()
			}
			wg.Wait()
			require.Equal(t, expected, result)
		}
	}
}

func BenchmarkTxOriginManager(b *testing.B) {
	for i := 0; i < b.N; i++ {
		partitions := uint64(10)
		clockMgr := NewTxClockManager(partitions)
		prtMgr := NewTxPartitionManager(partitions)
		originMgr := NewTxOriginManager(partitions, clockMgr, prtMgr)

		serviceA := "service-a"
		serviceB := "service-b"
		serviceC := "service-c"
		services := []string{serviceA, serviceB, serviceC}
		for _, service := range services {
			originMgr.Init(service)
		}

		concurrency := 10000
		r := rand.New(rand.NewSource(42))
		for partition := range partitions {
			for _, service := range services {
				result := []int{}
				perm := r.Perm(concurrency)
				var wg sync.WaitGroup
				wg.Add(concurrency)
				for _, ts := range perm {
					go func() {
						defer wg.Done()
						originMgr.Acquire(NewWaitMsg(partition, service, uint64(ts+1)))
						result = append(result, ts)
						originMgr.Release(partition, service)
					}()
				}
				wg.Wait()
			}
		}
	}
}
