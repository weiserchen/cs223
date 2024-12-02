package cc

import (
	"context"

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
	conn             *pgxpool.Pool
}

func NewTxManager(conn *pgxpool.Pool, partitions uint64, services []string) *TxManager {
	filterMgr := NewTxFilterManager(partitions)
	senderClockMgr := NewTxClockManager(partitions)
	receiverClockMgr := NewTxClockManager(partitions)
	senderPrtMgr := NewTxPartitionManager(partitions)
	receiverPrtMgr := NewTxPartitionManager(partitions)
	originMgr := NewTxOriginManager(partitions, receiverClockMgr, receiverPrtMgr)
	execMgr := NewTxExecutorManager()
	for _, service := range services {
		filterMgr.Init(service)
		originMgr.Init(service)
	}
	return &TxManager{
		SenderClockMgr:   senderClockMgr,
		ReceiverClockMgr: receiverClockMgr,
		SenderPrtMgr:     senderPrtMgr,
		ReceiverPrtMgr:   receiverPrtMgr,
		FilterMgr:        filterMgr,
		OriginMgr:        originMgr,
		ExecMgr:          execMgr,
	}
}

func (mgr *TxManager) Recover() error {
	ctx := context.Background()

	senderClockQuery := `
		SELECT prt, svc, ts
		FROM TxSenderClocks;
	`

	rows, err := mgr.conn.Query(ctx, senderClockQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var partition, timestamp uint64
		var service string
		if err = rows.Scan(&partition, &service, &timestamp); err != nil {
			return err
		}
		mgr.SenderClockMgr.Set(partition, service, timestamp)
	}

	receiverClockQuery := `
		SELECT prt, svc, ts
		FROM TxReceiverClocks;
	`

	rows, err = mgr.conn.Query(ctx, receiverClockQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var partition, timestamp uint64
		var service string
		if err = rows.Scan(&partition, &service, &timestamp); err != nil {
			return err
		}
		mgr.ReceiverClockMgr.Set(partition, service, timestamp)
	}

	// executorQuery := `
	//
	// `

	// resultQuery := `
	//
	// `

	return nil
}
