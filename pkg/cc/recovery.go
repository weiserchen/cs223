package cc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecoveryRequest = errors.New("failed to perform recovery request")
)

const (
	HeaderKeyExecCtx = "X-Tx-Executor-Context"
)

const (
	MaxRecoveryRetry     = 10
	RecoveryWaitTimeUnit = 1 * time.Millisecond
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
	// aborted = 2, completed = 5, rollback = 3
	// rollback is currently not supported
	executorQuery := `
		SELECT exec_id, checkpoint
		FROM TxExecutor
		WHERE status NOT IN (2, 3, 5);
	`
	rows, err := mgr.conn.Query(ctx, executorQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var reqs []*http.Request
	for rows.Next() {
		var checkpoint []byte
		var execID uint64
		if err = rows.Scan(&execID, &checkpoint); err != nil {
			return err
		}

		var execCtx *TxExecutorContext
		if err = json.Unmarshal(checkpoint, &execCtx); err != nil {
			return err
		}
		execCtx.ExecID = execID

		var b []byte
		b, err = json.Marshal(execCtx.Input)
		if err != nil {
			return errors.Join(err, format.ErrJsonEncode)
		}

		method, endpoint := execCtx.Method, execCtx.Endpoint
		req, err := http.NewRequest(method, endpoint, bytes.NewReader(b))
		req.Header.Add(HeaderKeyExecCtx, execCtx.Encode())
		if err != nil {
			return err
		}

		reqs = append(reqs, req)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	for _, req := range reqs {
		go recoveryRequest(client, req)
	}
	return nil
}

// Future Work: concurrent retry
// https://stackoverflow.com/questions/39791021/how-to-read-multiple-times-from-same-io-reader
// https://stackoverflow.com/questions/19929386/handling-connection-reset-errors-in-go
// https://stackoverflow.com/questions/37774624/go-http-get-concurrency-and-connection-reset-by-peer
func recoveryRequest(client *http.Client, req *http.Request) error {
	var resp *http.Response
	var err error

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	for i := range MaxRecoveryRetry {
		r := req.Clone(req.Context())
		r.Body = io.NopCloser(bytes.NewReader(bytes.Clone(body)))

		resp, err = client.Do(r)
		if err == nil {
			break
		}
		if i == MaxRecoveryRetry-1 {
			return err
		}
		sleepTime := time.Duration(rand.Int63n(10)) * RecoveryWaitTimeUnit
		log.Println("sleep:", sleepTime, err)
		time.Sleep(sleepTime)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return ErrRecoveryRequest
	}
	return nil
}
