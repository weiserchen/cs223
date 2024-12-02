package cc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxClockManager(t *testing.T) {
	partitions := uint64(10)
	clockMgr := NewTxClockManager(partitions)

	serviceA := "service-a"
	serviceB := "service-b"
	serviceC := "service-c"
	services := []string{serviceA, serviceB, serviceC}

	for _, service := range services {
		clockMgr.InitService(service)
	}

	for partition := range partitions {
		for _, service := range services {
			require.Equal(t, clockMgr.Get(partition, service), uint64(0))
		}
	}

	for partition := range partitions {
		for _, service := range services {
			clockMgr.Inc(partition, service)
			require.Equal(t, clockMgr.Get(partition, service), uint64(1))
		}
	}

	for partition := range partitions {
		for _, service := range services {
			clockMgr.Set(partition, service, uint64(100))
			require.Equal(t, clockMgr.Get(partition, service), uint64(100))
		}
	}
}
