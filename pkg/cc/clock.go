package cc

type TxClockManager struct {
	partitions uint64
	clocks     map[uint64]map[string]uint64
}

func NewTxClockManager(partitions uint64) *TxClockManager {
	partitions = GenPartitions(partitions)
	clocks := make(map[uint64]map[string]uint64)
	for partition := range partitions {
		clocks[partition] = make(map[string]uint64)
	}
	return &TxClockManager{
		partitions: GenPartitions(partitions),
		clocks:     clocks,
	}
}

func (mgr *TxClockManager) InitService(service string) {
	for partition := range mgr.partitions {
		mgr.clocks[partition][service] = 0
	}
}

func (mgr *TxClockManager) Get(partition uint64, service string) uint64 {
	return mgr.clocks[partition][service]
}

func (mgr *TxClockManager) Set(partition uint64, service string, timestamp uint64) {
	mgr.clocks[partition][service] = timestamp
}

func (mgr *TxClockManager) Inc(partition uint64, service string) {
	mgr.clocks[partition][service]++
}
