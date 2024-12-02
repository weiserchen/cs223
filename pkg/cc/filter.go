package cc

import (
	"errors"

	"github.com/emirpasic/gods/v2/sets/hashset"
)

var (
	ErrTxRequestDropped  = errors.New("tx request is dropped")
	ErrTxResponseDropped = errors.New("tx response is dropped")
)

type TxFilterType string
type TxFilterOp string

const (
	TxFilterTypeRequest  TxFilterType = "filter-request"
	TxFilterTypeResponse TxFilterType = "filter-response"

	TxFilterOpAdd    TxFilterOp = "filter-add"
	TxFilterOpRemove TxFilterOp = "filter-remove"
	TxFilterOpClear  TxFilterOp = "filter-clear"
)

type TxFilterManager struct {
	reqFilter  map[uint64]map[string]*hashset.Set[string]
	respFilter map[uint64]map[string]*hashset.Set[string]
	partitions uint64
}

func NewTxFilterManager(partitions uint64) *TxFilterManager {
	partitions = GenPartitions(partitions)
	reqFilter := map[uint64]map[string]*hashset.Set[string]{}
	respFilter := map[uint64]map[string]*hashset.Set[string]{}
	for partition := range partitions {
		reqFilter[partition] = make(map[string]*hashset.Set[string])
		respFilter[partition] = make(map[string]*hashset.Set[string])
	}
	return &TxFilterManager{
		partitions: partitions,
		reqFilter:  reqFilter,
		respFilter: respFilter,
	}
}

func (mgr *TxFilterManager) Init(service string) {
	for partition := range mgr.partitions {
		mgr.reqFilter[partition][service] = hashset.New[string]()
		mgr.respFilter[partition][service] = hashset.New[string]()
	}
}

func (mgr *TxFilterManager) AddReqFilter(partition uint64, service string, attrs []string) {
	partition = partition % mgr.partitions
	set := mgr.reqFilter[partition][service]
	set.Add(attrs...)
}

func (mgr *TxFilterManager) AddRespFilter(partition uint64, service string, attrs []string) {
	partition = partition % mgr.partitions
	set := mgr.respFilter[partition][service]
	set.Add(attrs...)
}

func (mgr *TxFilterManager) RemoveReqFilter(partition uint64, service string, attrs []string) {
	partition = partition % mgr.partitions
	set := mgr.reqFilter[partition][service]
	set.Remove(attrs...)
}

func (mgr *TxFilterManager) RemoveRespFilter(partition uint64, service string, attrs []string) {
	partition = partition % mgr.partitions
	set := mgr.respFilter[partition][service]
	set.Remove(attrs...)
}

func (mgr *TxFilterManager) ClearReqFilter(partition uint64, service string) {
	partition = partition % mgr.partitions
	set := mgr.reqFilter[partition][service]
	set.Clear()
}

func (mgr *TxFilterManager) ClearRespFilter(partition uint64, service string) {
	partition = partition % mgr.partitions
	set := mgr.respFilter[partition][service]
	set.Clear()
}

func (mgr *TxFilterManager) DropReq(partition uint64, service string, attrs []string) bool {
	partition = partition % mgr.partitions
	set := mgr.reqFilter[partition][service]
	return set.Contains(attrs...)
}

func (mgr *TxFilterManager) DropResp(partition uint64, service string, attrs []string) bool {
	partition = partition % mgr.partitions
	set := mgr.respFilter[partition][service]
	return set.Contains(attrs...)
}
