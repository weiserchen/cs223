package cc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"txchain/pkg/database"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestHighRequestPerSecond(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	mux := http.NewServeMux()
	httpMethod := http.MethodPost
	httpPath := "/concurrency"
	pattern := fmt.Sprintf("%s %s", httpMethod, httpPath)
	mux.Handle(pattern, handler)

	timeout := 10 * time.Second
	testServer := httptest.NewUnstartedServer(mux)
	testServer.Config.ReadHeaderTimeout = timeout
	testServer.Config.ReadTimeout = timeout
	testServer.Config.WriteTimeout = timeout
	testServer.Config.IdleTimeout = timeout
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	// listener = netutil.LimitListener(listener, 10000)
	testServer.Listener = listener
	testServer.Start()

	addr := testServer.URL + httpPath
	concurrency := 10000

	transport := &http.Transport{
		MaxIdleConns:        10000,
		MaxIdleConnsPerHost: 10000,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Int63n(1000)) * time.Millisecond)
			s := "abc"
			req, err := http.NewRequest(httpMethod, addr, strings.NewReader(s))
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			_, _ = io.ReadAll(resp.Body)
		}()
	}
	wg.Wait()
}

func TestTxRecovery(t *testing.T) {
	pgc, err := database.NewContainerTablesTx(t, "17.1")
	defer func() {
		if pgc != nil {
			testcontainers.CleanupContainer(t, pgc.Container)
		}
	}()
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, pgc.Endpoint())
	require.NoError(t, err)

	transport := http.DefaultTransport.(*http.Transport)
	transport.MaxIdleConns = 0

	partitions := uint64(20)
	// services := []string{"service-a"}
	services := []string{"service-a", "service-b", "service-c"}
	sendClockMgr := NewTxClockManager(partitions)
	recvClockMgr := NewTxClockManager(partitions)
	execMgr := NewTxExecutorManager(ExponentialBackoffRetry(100 * time.Millisecond))
	recoveryMgr := NewTxRecoveryManager(conn, sendClockMgr, recvClockMgr, execMgr)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stages := defaultStages()
		stage1 := stages[execStage1Recovery]
		stage2 := stages[execStage2Recovery]
		stage3 := stages[execStage3Recovery]
		checkpointer := DefaultCheckpointer(conn)
		encodedExecCtx := r.Header.Get(HeaderKeyExecCtx)
		execCtx, err := DecodeTxExecutorContext(encodedExecCtx)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(format.ErrJsonDecode, err), http.StatusBadRequest)
			return
		}
		executor := NewTxExecutor(execCtx, checkpointer)
		executor.
			CommitStage(stage1).
			Stage(stage2).
			Stage(stage3)

		_, err = executor.Run()
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrRecoveryRequest, err), http.StatusInternalServerError)
			return
		}

		execMgr.Send(executor)
	})
	mux := http.NewServeMux()
	httpMethod := http.MethodPost
	httpPath := "/test/recovery"
	pattern := fmt.Sprintf("%s %s", httpMethod, httpPath)
	mux.Handle(pattern, handler)
	testServer := httptest.NewServer(mux)
	testURL := testServer.URL + httpPath
	concurrency := 20

	testInsertSendRecvClocks(t, conn, partitions, services)
	testInsertPartialExecutors(t, conn, partitions, services, concurrency, testURL)

	go execMgr.Run()
	err = recoveryMgr.Recover()
	require.NoError(t, err)

	// check send clocks
	for prt := range partitions {
		for _, svc := range services {
			ts := sendClockMgr.Get(prt, svc)
			require.Equal(t, prt+10, ts)
		}
	}

	// check recv clocks
	for prt := range partitions {
		for _, svc := range services {
			ts := recvClockMgr.Get(prt, svc)
			require.Equal(t, prt+10, ts)
		}
	}

	reqCount := int(partitions) * len(services) * concurrency
	// pending, committed, force complete, completed
	time.Sleep(time.Second)
	execCtxs := testAllExecutor(t, conn, reqCount*4, ExecStatusCompleted)
	// pending, committed, completed -> 1 + 2 + 3 = 6
	// force complete -> 1 + 2 + 0 = 3
	testSumEqual(t, execCtxs, reqCount*(6*3+3))
}

func testInsertSendRecvClocks(
	t *testing.T,
	conn *pgxpool.Pool,
	partitions uint64,
	services []string,
) {
	ctx := context.Background()
	sendClockQuery := `
		INSERT INTO TxSenderClocks (prt, svc, ts) 
		VALUES (@partition, @service, @timestamp);
	`
	recvClockQuery := `
		INSERT INTO TxReceiverClocks (prt, svc, ts) 
		VALUES (@partition, @service, @timestamp);
	`

	batch := &pgx.Batch{}
	for prt := range partitions {
		for _, svc := range services {
			args := pgx.NamedArgs{
				"partition": prt,
				"service":   svc,
				"timestamp": prt + 10,
			}
			batch.Queue(sendClockQuery, args)
			batch.Queue(recvClockQuery, args)
		}
	}

	results := conn.SendBatch(ctx, batch)
	defer results.Close()

	for range partitions {
		for range services {
			_, err := results.Exec()
			require.NoError(t, err)
		}
	}
}

func testInsertPartialExecutors(
	t *testing.T,
	conn *pgxpool.Pool,
	partitions uint64,
	services []string,
	concurrency int,
	endpoint string,
) {
	ctx := context.Background()
	executorQuery := `
		INSERT INTO TxExecutor (status, checkpoint) 
		VALUES (@status, @checkpoint);
	`

	queryCount := 0
	batch := &pgx.Batch{}
	for prt := range partitions {
		for _, svc := range services {
			for range concurrency {
				execCtx := defaultExecCtx()
				execCtx.CtrlCtx.Partition = prt
				execCtx.CtrlCtx.Service = svc
				execCtx.Method = http.MethodPost
				execCtx.Endpoint = endpoint

				// aborted
				execCtx.Status = ExecStatusAborted
				execCtx.CtrlCtx.Attrs = []string{"aborted"}
				b, err := json.Marshal(execCtx)
				require.NoError(t, err)
				args := pgx.NamedArgs{
					"status":     execCtx.Status,
					"checkpoint": b,
				}
				batch.Queue(executorQuery, args)
				queryCount++

				// completed
				execCtx.Status = ExecStatusCompleted
				execCtx.CtrlCtx.Attrs = []string{"completed"}
				execCtx.Input = Input{
					Value: []int{1, 2, 3},
				}
				execCtx.Result = Result{1}
				b, err = json.Marshal(execCtx)
				require.NoError(t, err)
				args = pgx.NamedArgs{
					"status":     execCtx.Status,
					"checkpoint": b,
				}
				batch.Queue(executorQuery, args)
				queryCount++

				// pending
				execCtx.Status = ExecStatusPending
				execCtx.CtrlCtx.Attrs = []string{"pending"}
				execCtx.Input = Input{
					Value: []int{},
				}
				execCtx.Result = nil
				b, err = json.Marshal(execCtx)
				require.NoError(t, err)
				args = pgx.NamedArgs{
					"status":     execCtx.Status,
					"checkpoint": b,
				}
				batch.Queue(executorQuery, args)
				queryCount++

				// committed
				execCtx.Status = ExecStatusCommitted
				execCtx.CtrlCtx.Attrs = []string{"committed"}
				execCtx.Input = Input{
					Value: []int{1},
				}
				execCtx.Result = Result{1}
				b, err = json.Marshal(execCtx)
				require.NoError(t, err)
				args = pgx.NamedArgs{
					"status":     execCtx.Status,
					"checkpoint": b,
				}
				batch.Queue(executorQuery, args)
				queryCount++

				// force complete
				execCtx.Status = ExecStatusForceComplete
				execCtx.CtrlCtx.Attrs = []string{"force complete"}
				execCtx.Input = Input{
					Value: []int{1, 2},
				}
				execCtx.Result = Result{1}
				execCtx.Curr = 1
				b, err = json.Marshal(execCtx)
				require.NoError(t, err)
				args = pgx.NamedArgs{
					"status":     execCtx.Status,
					"checkpoint": b,
				}
				batch.Queue(executorQuery, args)
				queryCount++
			}
		}
	}

	results := conn.SendBatch(ctx, batch)
	defer results.Close()

	for range queryCount {
		_, err := results.Exec()
		require.NoError(t, err)
	}

	countQuery := `
		SELECT COUNT(*)
		FROM TxExecutor
		WHERE status = @status;
	`
	uncompletedStatus := []ExecStatus{ExecStatusPending, ExecStatusCommitted, ExecStatusForceComplete}
	for _, status := range uncompletedStatus {
		var count int
		args := pgx.NamedArgs{
			"status": status,
		}
		row := conn.QueryRow(ctx, countQuery, args)
		err := row.Scan(&count)
		require.NoError(t, err)
		require.Equal(t, int(partitions)*len(services)*concurrency, count)
	}
}
