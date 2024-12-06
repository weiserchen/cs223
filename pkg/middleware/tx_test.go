package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
	"txchain/pkg/cc"
	"txchain/pkg/database"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

type Input struct {
	Value uint64 `json:"value"`
}

var _ cc.Partition = (*Input)(nil)

func NewInput(v uint64) *Input {
	return &Input{
		Value: v,
	}
}

func (input *Input) Keys() []any {
	return []any{input.Value}
}

type Result struct {
	Result any `json:"result"`
}

func TestTxMiddlewares(t *testing.T) {
	var txMgr *cc.TxManager
	var serverTx, serverA, serverB, serverC *httptest.Server
	var addrTx, addrA, addrB, addrC string
	var connTx *pgxpool.Pool
	type APIService int

	const api APIService = 0
	partitions := uint64(10)
	concurrency := uint64(10)
	serviceTx := "service-tx"
	serviceA := "service-a"
	serviceB := "service-b"
	serviceC := "service-c"

	serverTxHTTPMethod := http.MethodPost
	serverTxHTTPPath := "/tx"
	serverTxHTTPPattern := fmt.Sprintf("%s %s", serverTxHTTPMethod, serverTxHTTPPath)

	serverAHTTPMethod := http.MethodPost
	serverAHTTPPath := "/a"
	serverAHTTPPattern := fmt.Sprintf("%s %s", serverAHTTPMethod, serverAHTTPPath)

	serverBHTTPMethod := http.MethodPost
	serverBHTTPPath := "/b"
	serverBHTTPPattern := fmt.Sprintf("%s %s", serverBHTTPMethod, serverBHTTPPath)

	serverCHTTPMethod := http.MethodPost
	serverCHTTPPath := "/c"
	serverCHTTPPattern := fmt.Sprintf("%s %s", serverCHTTPMethod, serverCHTTPPath)

	transport := &http.Transport{
		MaxIdleConns:        10000,
		MaxIdleConnsPerHost: 10000,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	traceRecorderA := cc.NewTraceRecorder(partitions)
	traceRecorderB := cc.NewTraceRecorder(partitions)
	traceRecorderC := cc.NewTraceRecorder(partitions)
	traceRecorderTx := cc.NewTraceRecorder(partitions)

	logger := NewDebugLogger()

	{
		_, conn, close := initServer(t)
		defer close()

		txMgr = cc.NewTxManager(conn, partitions, []string{serviceA, serviceTx})
		txMgr.Instrumenter.Recorder(traceRecorderA)
		middlewares := []Middlerware{
			TxParticipant(txMgr, logger, serviceA),
			ValidateBody[Input],
		}

		handler := serverHandler(conn, api)
		mux := http.NewServeMux()
		mux.Handle(serverAHTTPPattern, Chain(handler, middlewares...))
		serverA = httptest.NewServer(mux)
		addrA = serverA.URL + serverAHTTPPath
		log.Println(addrA)
		defer serverA.Close()
	}
	{
		_, conn, close := initServer(t)
		defer close()

		txMgr = cc.NewTxManager(conn, partitions, []string{serviceB, serviceTx})
		txMgr.Instrumenter.Recorder(traceRecorderB)
		middlewares := []Middlerware{
			TxParticipant(txMgr, logger, serviceB),
			ValidateBody[Input],
		}

		handler := serverHandler(conn, api)
		mux := http.NewServeMux()
		mux.Handle(serverBHTTPPattern, Chain(handler, middlewares...))
		serverB = httptest.NewServer(mux)
		addrB = serverB.URL + serverBHTTPPath
		log.Println(addrB)
		defer serverB.Close()
	}
	{
		_, conn, close := initServer(t)
		defer close()

		txMgr = cc.NewTxManager(conn, partitions, []string{serviceC, serviceTx})
		txMgr.Instrumenter.Recorder(traceRecorderC)
		middlewares := []Middlerware{
			TxParticipant(txMgr, logger, serviceC),
			ValidateBody[Input],
		}

		handler := serverHandler(conn, api)
		mux := http.NewServeMux()
		mux.Handle(serverCHTTPPattern, Chain(handler, middlewares...))
		serverC = httptest.NewServer(mux)
		addrC = serverC.URL + serverCHTTPPath
		log.Println(addrC)
		defer serverC.Close()
	}
	{
		_, conn, close := initServer(t)
		defer close()
		connTx = conn

		txMgr = cc.NewTxManager(conn, partitions, []string{serviceTx})
		txMgr.Instrumenter.Recorder(traceRecorderTx)
		go txMgr.ExecMgr.Run()

		middlewares := []Middlerware{
			ValidateBody[*Input],
			TxCoordinator[*Input](conn, txMgr, logger, serviceTx, []string{serviceA, serviceB, serviceC}),
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			execCtx, ok := cc.GetTxExecCtx(ctx)
			if !ok {
				format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxExecEmpty, nil), http.StatusBadRequest)
				return
			}

			timestamps := execCtx.Timestamps
			ctrlCtx := execCtx.CtrlCtx
			dryRun := execCtx.Recovered && execCtx.Status == cc.ExecStatusPending
			stageA := cc.NewExecutorStage()
			stageA.Stage(httpStageFunc(client, ctrlCtx, serverAHTTPMethod, addrA, timestamps[0], dryRun))
			stageB := cc.NewExecutorStage()
			stageB.Stage(httpStageFunc(client, ctrlCtx, serverBHTTPMethod, addrB, timestamps[1], false))
			stageC := cc.NewExecutorStage()
			stageC.Stage(httpStageFunc(client, ctrlCtx, serverCHTTPMethod, addrC, timestamps[2], false))
			checkpointer := cc.DefaultCheckpointer(conn)

			executor := cc.NewTxExecutor(execCtx, checkpointer)
			executor.
				CommitStage(stageA).
				Stage(stageB).
				Stage(stageC)

			res, err := executor.Run()
			log.Println(res)
			if err != nil {
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxExecutor, err), http.StatusInternalServerError)
				return
			}

			_ = executor.Checkpoint()

			txMgr.ExecMgr.Send(executor)
			result := Result{
				Result: res,
			}
			format.WriteJsonResponse(w, result, http.StatusOK)
		})

		mux := http.NewServeMux()
		mux.Handle(serverTxHTTPPattern, Chain(handler, middlewares...))
		serverTx = httptest.NewServer(mux)
		addrTx = serverTx.URL + serverTxHTTPPath
		log.Println(addrTx)
		defer serverTx.Close()
	}

	var wg sync.WaitGroup
	for prt := range partitions {
		for i := range concurrency {
			wg.Add(1)
			run := func() {
				defer wg.Done()
				ts := prt*partitions + i + 1
				input := Input{
					Value: ts,
				}

				b, err := json.Marshal(input)
				require.NoError(t, err)

				ctrlCtx := &cc.TxControlContext{}
				ctrlCtxEncoded := ctrlCtx.Encode()

				req, err := http.NewRequest(serverTxHTTPMethod, addrTx, bytes.NewReader(b))
				require.NoError(t, err)
				req.Header.Add(headerTxControlContext, ctrlCtxEncoded)
				req.Header.Add(headerTxLoggerID, strconv.Itoa(int(i)+1))

				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				log.Println(resp.StatusCode)
				if resp.StatusCode >= 300 {
					body, err := io.ReadAll(resp.Body)
					require.NoError(t, err)
					var errResp format.ErrorResponse
					err = json.Unmarshal(body, &errResp)
					require.NoError(t, err)
					t.Log(errResp)
					t.Fatal()
				}

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				log.Println(string(body))

				var result Result
				err = json.Unmarshal(body, &result)
				require.NoError(t, err, string(body))

				value, err := format.UnmarshalInput[Input](result.Result)
				require.NoError(t, err)
				require.Equal(t, Input{ts}, value)
			}

			go run()
			// run()
		}
	}
	wg.Wait()

	sendOrder := make([][]uint64, 10)
	recvOrderA := make([][]uint64, 10)
	recvOrderB := make([][]uint64, 10)
	recvOrderC := make([][]uint64, 10)

	require.Eventually(t, func() bool {
		completedCount := 0
		for prt := range partitions {
			for _, resp := range traceRecorderTx.GetReq(prt) {
				value := resp.(*Input)
				sendOrder[prt] = append(sendOrder[prt], value.Value)
			}
			for _, resp := range traceRecorderA.GetResp(prt) {
				value := resp.(database.Result[uint64])
				recvOrderA[prt] = append(recvOrderA[prt], value.Value)
			}
			for _, resp := range traceRecorderB.GetResp(prt) {
				value := resp.(database.Result[uint64])
				recvOrderB[prt] = append(recvOrderB[prt], value.Value)
			}
			for _, resp := range traceRecorderC.GetResp(prt) {
				value := resp.(database.Result[uint64])
				recvOrderC[prt] = append(recvOrderC[prt], value.Value)
			}
			log.Printf(
				"prt: %d send: %v recvA: %v recvB: %v recvC: %v",
				prt,
				sendOrder[prt],
				recvOrderA[prt],
				recvOrderB[prt],
				recvOrderC[prt],
			)
			completedCount += len(recvOrderC[prt])
		}
		return completedCount == int(partitions*concurrency)
	}, 10*time.Second, 500*time.Millisecond)

	for i := range concurrency {
		id := strconv.Itoa(int(i) + 1)
		logger.Print(id, log.Writer())
	}

	require.Equal(t, sendOrder, recvOrderA)
	require.Equal(t, recvOrderA, recvOrderB)
	require.Equal(t, recvOrderB, recvOrderC)

	totalCount := int(partitions * concurrency)
	testAllExecutor(t, connTx, totalCount, cc.ExecStatusCompleted, time.Second)
}

func serverHandler[API comparable](
	conn *pgxpool.Pool,
	api API,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		table := database.NewTxTable[API](conn)
		table.BeforeHook(api, cc.TxDedupBeforeHook)
		table.AfterHook(api, cc.TxDedupBeforeHook)

		req := UnmarshalRequest[Input](r)
		database.UnwrapResult(r.Context(), func(ctx context.Context) (uint64, error) {
			lifecycle := database.NewTxLifeCycle[API, uint64](table)
			return lifecycle.Start(api, ctx, func(ctx context.Context, tx pgx.Tx) (uint64, error) {
				return req.Value, nil
			})
		})
	})
}

func initServer(t *testing.T) (
	pgc *database.PgContainer,
	conn *pgxpool.Pool,
	close func(),
) {
	var err error
	version := "17.1"

	pgc = &database.PgContainer{}
	pgc, err = database.NewContainerTablesTx(t, version)
	defer func() {
		if err != nil {
			testcontainers.CleanupContainer(t, pgc.Container)
		}
	}()

	close = func() {
		testcontainers.CleanupContainer(t, pgc.Container)
	}
	require.NoError(t, err)

	ctx := context.Background()
	conn, err = pgxpool.New(ctx, pgc.Endpoint())
	require.NoError(t, err)

	return pgc, conn, close
}

func httpStageFunc(
	client *http.Client,
	ctrlCtx *cc.TxControlContext,
	method, addr string,
	timestamp uint64,
	dryRun bool,
) cc.StageFunc {
	return func(input any) (result any, output any, err error) {
		var s Input
		var ok bool
		var b []byte
		var req *http.Request
		var resp *http.Response
		s, ok = input.(Input)
		if !ok {
			s, err = format.UnmarshalInput[Input](input)
			if err != nil {
				return nil, nil, err
			}
		}

		b, err = json.Marshal(s)
		if err != nil {
			return nil, nil, err
		}

		req, err = http.NewRequest(method, addr, bytes.NewReader(b))
		if err != nil {
			return nil, nil, err
		}

		stageCtx := &cc.TxStageContext{
			Partition: ctrlCtx.Partition,
			Service:   ctrlCtx.Service,
			Timestamp: timestamp,
			Attrs:     ctrlCtx.Attrs,
			DryRun:    dryRun,
		}
		req.Header.Add(headerTxLoggerID, ctrlCtx.LoggerID)
		req.Header.Add(headerTxStageContext, stageCtx.Encode())

		resp, err = client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, nil, err
		}
		return s, s, nil
	}
}

func testAllExecutor(
	t *testing.T,
	conn *pgxpool.Pool,
	count int,
	status cc.ExecStatus,
	timeout time.Duration,
) {
	t.Helper()

	var statusCount int
	var err error
	require.Eventually(t, func() bool {
		query := `
			SELECT COUNT(*)
			FROM TxExecutor
			WHERE status = $1;
		`
		row := conn.QueryRow(context.Background(), query, status)
		if err = row.Scan(&statusCount); err != nil {
			return false
		}
		return statusCount == count
	}, timeout, 500*time.Millisecond, statusCount)
}
