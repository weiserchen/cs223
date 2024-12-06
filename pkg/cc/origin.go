package cc

import (
	pq "github.com/emirpasic/gods/v2/queues/priorityqueue"
)

type WaitMsg struct {
	partition uint64
	service   string
	timestamp uint64
	reply     chan struct{}
}

func NewWaitMsg(
	partition uint64,
	service string,
	timestamp uint64,
) WaitMsg {
	reply := make(chan struct{})
	return WaitMsg{
		partition: partition,
		service:   service,
		timestamp: timestamp,
		reply:     reply,
	}
}

// sort ascending
func timestampComparator(a, b WaitMsg) int {
	timestampA := a.timestamp
	timestampB := b.timestamp
	switch {
	case timestampA > timestampB:
		return 1
	case timestampA < timestampB:
		return -1
	default:
		return 0
	}
}

type TxOriginManager struct {
	partitions uint64
	queues     map[uint64]map[string]*pq.Queue[WaitMsg]
	// receiver clocks
	clockMgr *TxClockManager
	// receiver partitions
	prtMgr *TxPartitionManager
}

func NewTxOriginManager(
	partitions uint64,
	receiverClockMgr *TxClockManager,
	receiverPrtMgr *TxPartitionManager,
) *TxOriginManager {
	partitions = GenPartitions(partitions)
	queues := make(map[uint64]map[string]*pq.Queue[WaitMsg])
	for partition := range partitions {
		queues[partition] = make(map[string]*pq.Queue[WaitMsg])
	}
	return &TxOriginManager{
		partitions: partitions,
		queues:     queues,
		clockMgr:   receiverClockMgr,
		prtMgr:     receiverPrtMgr,
	}
}

func (mgr *TxOriginManager) Init(service string) {
	for partition := range mgr.partitions {
		mgr.queues[partition][service] = pq.NewWith(timestampComparator)
	}
}

func (mgr *TxOriginManager) Acquire(msg WaitMsg) bool {
	ok := mgr.enqueue(msg)
	if ok {
		mgr.next(msg.partition, msg.service)
		<-msg.reply
	}
	return ok
}

func (mgr *TxOriginManager) enqueue(msg WaitMsg) bool {
	partition := msg.partition
	service := msg.service
	timestamp := msg.timestamp

	mgr.prtMgr.Lock(partition)
	defer mgr.prtMgr.Unlock(partition)

	currTs := mgr.clockMgr.Get(partition, service)
	if timestamp <= currTs {
		return false
	}
	mgr.queues[partition][service].Enqueue(msg)
	return true
}

func (mgr *TxOriginManager) Release(partition uint64, service string) {
	mgr.advance(partition, service)
	mgr.next(partition, service)
}

func (mgr *TxOriginManager) advance(partition uint64, service string) {
	mgr.prtMgr.Lock(partition)
	defer mgr.prtMgr.Unlock(partition)
	mgr.clockMgr.Inc(partition, service)
}

func (mgr *TxOriginManager) next(partition uint64, service string) {
	mgr.prtMgr.Lock(partition)
	defer mgr.prtMgr.Unlock(partition)

	var topMsg WaitMsg
	var ok bool

	q := mgr.queues[partition][service]
	// log.Println("origin queue:", q.Values())
	topMsg, ok = q.Peek()
	if !ok {
		return
	}

	currTs := mgr.clockMgr.Get(partition, service)
	nextTs := currTs + 1
	if topMsg.timestamp == nextTs {
		_, _ = q.Dequeue()
		go func() {
			topMsg.reply <- struct{}{}
		}()
	}
}
