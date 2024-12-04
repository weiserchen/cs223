package cc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecoveryRequest = errors.New("failed to perform recovery request")
)

type TxRecoveryManager struct {
	conn         *pgxpool.Pool
	sendClockMgr *TxClockManager
	recvClockMgr *TxClockManager
	execMgr      *TxExecutorManager
}

func NewTxRecoveryManager(
	conn *pgxpool.Pool,
	sendClockMgr *TxClockManager,
	recvClockMgr *TxClockManager,
	execMgr *TxExecutorManager,
) *TxRecoveryManager {
	return &TxRecoveryManager{
		conn:         conn,
		sendClockMgr: sendClockMgr,
		recvClockMgr: recvClockMgr,
		execMgr:      execMgr,
	}
}

// it should be called AFTER the server is running
func (mgr *TxRecoveryManager) Recover() error {
	var err error
	err = mgr.recoverSendClocks()
	if err != nil {
		return err
	}

	err = mgr.recoverRecvClocks()
	if err != nil {
		return err
	}

	err = mgr.recoverExecutors()
	if err != nil {
		return err
	}
	return nil
}

func (mgr *TxRecoveryManager) recoverSendClocks() error {
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
		mgr.sendClockMgr.Set(partition, service, timestamp)
	}
	return nil
}

func (mgr *TxRecoveryManager) recoverRecvClocks() error {
	ctx := context.Background()
	receiverClockQuery := `
		SELECT prt, svc, ts
		FROM TxReceiverClocks;
	`

	rows, err := mgr.conn.Query(ctx, receiverClockQuery)
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
		mgr.recvClockMgr.Set(partition, service, timestamp)
	}
	return nil
}

func (mgr *TxRecoveryManager) recoverExecutors() error {
	ctx := context.Background()
	// aborted = 2, completed = 5
	executorQuery := `
		SELECT checkpoint
		FROM TxExecutor
		WHERE status NOT IN (2, 5);
	`
	rows, err := mgr.conn.Query(ctx, executorQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var reqs []*http.Request
	for rows.Next() {
		var checkpoint []byte
		if err = rows.Scan(&checkpoint); err != nil {
			return err
		}

		var execCtx *TxExecutorContext
		if err = json.Unmarshal(checkpoint, &execCtx); err != nil {
			return err
		}

		var b []byte
		b, err = json.Marshal(execCtx.Input)
		if err != nil {
			return errors.Join(err, format.ErrJsonEncode)
		}

		method, endpoint := execCtx.Method, execCtx.Endpoint
		req, err := http.NewRequest(method, endpoint, bytes.NewReader(b))
		if err != nil {
			return err
		}

		reqs = append(reqs, req)
	}

	for _, req := range reqs {
		go recoveryRequest(req)
	}
	return nil
}

// Future Work: create a error handler for retry
func recoveryRequest(req *http.Request) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return ErrRecoveryRequest
	}
	return nil
}
