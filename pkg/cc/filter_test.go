package cc

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxFilterManager(t *testing.T) {
	partitions := uint64(10)
	serviceA := "service-a"
	serviceB := "service-b"
	serviceC := "service-c"
	services := []string{serviceA, serviceB, serviceC}

	filterMgr := NewTxFilterManager(uint64(partitions))

	for _, service := range services {
		filterMgr.Init(service)
	}

	// Check initial state
	dummyAttrs := []string{"dummy"}

	for partition := range partitions {
		for _, service := range services {
			require.False(t, filterMgr.DropReq(partition, service, dummyAttrs))
			require.False(t, filterMgr.DropResp(partition, service, dummyAttrs))
		}
	}

	// Check add req/resp filter
	filterServices := []string{serviceA, serviceB}
	filterPartitions := []uint64{3, 5, 9}
	reqAttr1, reqAttr2 := "apple", "banana"
	filterReqAttrs := []string{reqAttr1, reqAttr2}
	respAttr1, respAttr2 := "orange", "kiwi"
	filterRespAttrs := []string{respAttr1, respAttr2}
	for _, partition := range filterPartitions {
		for _, service := range filterServices {
			filterMgr.AddReqFilter(partition, service, filterReqAttrs)
			filterMgr.AddRespFilter(partition, service, filterRespAttrs)
		}
	}

	for partition := range partitions {
		for _, service := range services {
			isFilterService := slices.Contains(filterServices, service)
			isFilterPartition := slices.Contains(filterPartitions, partition)
			if isFilterService && isFilterPartition {
				require.False(t, filterMgr.DropReq(partition, service, dummyAttrs))
				require.True(t, filterMgr.DropReq(partition, service, []string{reqAttr1}))
				require.True(t, filterMgr.DropReq(partition, service, []string{reqAttr2}))
				require.True(t, filterMgr.DropReq(partition, service, []string{reqAttr1, reqAttr2}))
				require.False(t, filterMgr.DropResp(partition, service, dummyAttrs))
				require.True(t, filterMgr.DropResp(partition, service, []string{respAttr1}))
				require.True(t, filterMgr.DropResp(partition, service, []string{respAttr2}))
				require.True(t, filterMgr.DropResp(partition, service, []string{respAttr1, respAttr2}))
			} else {
				require.False(t, filterMgr.DropReq(partition, service, filterReqAttrs))
				require.False(t, filterMgr.DropResp(partition, service, filterRespAttrs))
			}
		}
	}

	// TODO: Check remove req/resp filter

	// Clear all filters
	for partition := range partitions {
		for _, service := range services {
			filterMgr.ClearReqFilter(partition, service)
			filterMgr.ClearRespFilter(partition, service)
		}
	}

	// Check again
	for _, service := range services {
		for partition := range partitions {
			require.False(t, filterMgr.DropReq(partition, service, dummyAttrs))
			require.False(t, filterMgr.DropReq(partition, service, filterReqAttrs))
			require.False(t, filterMgr.DropResp(partition, service, dummyAttrs))
			require.False(t, filterMgr.DropResp(partition, service, filterRespAttrs))
		}
	}
}
