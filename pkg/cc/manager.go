package cc

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager struct {
	SenderClockMgr   *TxClockManager
	ReceiverClockMgr *TxClockManager
	SenderPrtMgr     *TxPartitionManager
	ReceiverPrtMgr   *TxPartitionManager
	FilterMgr        *TxFilterManager
	OriginMgr        *TxOriginManager
	ExecMgr          *TxExecutorManager
	RecoveryMgr      *TxRecoveryManager
	Instrumenter     *TxInstrumenter
	conn             *pgxpool.Pool
}

func NewTxManager(conn *pgxpool.Pool, partitions uint64, services []string) *TxManager {
	filterMgr := NewTxFilterManager(partitions)
	senderClockMgr := NewTxClockManager(partitions)
	receiverClockMgr := NewTxClockManager(partitions)
	senderPrtMgr := NewTxPartitionManager(partitions)
	receiverPrtMgr := NewTxPartitionManager(partitions)
	originMgr := NewTxOriginManager(partitions, receiverClockMgr, receiverPrtMgr)
	execMgr := NewTxExecutorManager(ExponentialBackoffRetry(time.Second))
	recoveryMgr := NewTxRecoveryManager(conn, senderClockMgr, receiverClockMgr, execMgr)
	for _, service := range services {
		filterMgr.Init(service)
		originMgr.Init(service)
	}
	instrumenter := NewTxInstrumenter()
	return &TxManager{
		SenderClockMgr:   senderClockMgr,
		ReceiverClockMgr: receiverClockMgr,
		SenderPrtMgr:     senderPrtMgr,
		ReceiverPrtMgr:   receiverPrtMgr,
		FilterMgr:        filterMgr,
		OriginMgr:        originMgr,
		ExecMgr:          execMgr,
		RecoveryMgr:      recoveryMgr,
		Instrumenter:     instrumenter,
		conn:             conn,
	}
}
