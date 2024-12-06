package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"txchain/pkg/cc"
	"txchain/pkg/database"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMiddlewareTxGuard           = errors.New("failed to acuquire tx lock")
	ErrMiddlewareTxExecutor        = errors.New("failed to create tx executor")
	ErrMiddlewareTxMiddsingCtrlCtx = errors.New("missing control context")
)

func TxParticipant(mgr *cc.TxManager, logger Logger, participant string) Middlerware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if logger == nil {
				logger = &NopLogger{}
			}

			loggerID := r.Header.Get(headerTxLoggerID)
			if loggerID == "" {
				loggerID = DefaultLoggerID
			}

			session := logger.Session(loggerID)
			defer session.Done()

			traceCtx := format.NewTraceContext()
			ctx := format.SetTraceContext(r.Context(), traceCtx)

			session.Log("Participant: %s", participant)
			encoded := r.Header.Get(headerTxStageContext)
			// No tx stage context
			if encoded == "" {
				session.Log("No Stage Ctx")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			stageCtx, err := cc.DecodeTxStageContext(encoded)
			if err != nil {
				format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxStageContextDecode, err), http.StatusBadRequest)
				return
			}

			ctx = cc.SetTxStageCtx(ctx, stageCtx)

			session.Log("Stage Ctx: %v", stageCtx)
			partition := stageCtx.Partition
			service := stageCtx.Service
			timestamp := stageCtx.Timestamp
			attrs := stageCtx.Attrs
			filterMgr := mgr.FilterMgr
			dropReq := filterMgr.DropReq(partition, service, attrs)
			dropResp := filterMgr.DropResp(partition, service, attrs)

			session.Log("Drop Filter: req(%v) resp(%v)", dropReq, dropResp)
			if dropReq {
				session.Log("Drop Request")
				format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxRequestDropped, nil), http.StatusServiceUnavailable)
				return
			}

			writer := w
			if dropResp {
				writer = httptest.NewRecorder()
			}

			session.Log("Lock Partition: %d", partition)
			originMgr := mgr.OriginMgr
			originMgr.Acquire(cc.NewWaitMsg(partition, service, timestamp))
			session.Log("Origin TS: %d", timestamp)
			defer originMgr.Release(partition, service)

			recorder := mgr.Instrumenter

			recorder.VisitBefore(ctx)
			next.ServeHTTP(writer, r.WithContext(ctx))
			session.Log("Call Visit After")
			recorder.VisitAfter(ctx)

			if dropResp {
				format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxResponseDropped, nil), http.StatusServiceUnavailable)
				return
			}
		})
	}
}

func TxCoordinator[T cc.Partition](
	conn *pgxpool.Pool,
	mgr *cc.TxManager,
	logger Logger,
	service string,
	receivers []string,
) Middlerware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if logger == nil {
				logger = &NopLogger{}
			}

			loggerID := r.Header.Get(headerTxLoggerID)
			if loggerID == "" {
				loggerID = DefaultLoggerID
			}

			session := logger.Session(loggerID)
			defer session.Done()

			session.Log("Coordinator:")

			var execCtx *cc.TxExecutorContext
			var ctrlCtx *cc.TxControlContext
			var err error
			var ctx context.Context

			prtMgr := mgr.SenderPrtMgr
			clockMgr := mgr.SenderClockMgr
			ctx = r.Context()

			encodedExecCtx := r.Header.Get(headerTxExecutorContext)
			// Recovery request
			if encodedExecCtx != "" {
				session.Log("Recovery Request:")
				execCtx, err = cc.DecodeTxExecutorContext(encodedExecCtx)
				if err != nil {
					format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxExecutor, err), http.StatusBadRequest)
					return
				}
				execCtx.Recovered = true
				ctx = cc.SetTxExecCtx(ctx, execCtx)
			}

			if execCtx != nil {
				partition := execCtx.CtrlCtx.Partition
				prtMgr.Lock(partition)
				defer prtMgr.Unlock(partition)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// New request
			encodedCtrlCtx := r.Header.Get(headerTxControlContext)
			if encodedCtrlCtx == "" {
				ctrlCtx = &cc.TxControlContext{}
			} else {
				ctrlCtx, err = cc.DecodeTxControlContext(encodedCtrlCtx)
				if err != nil {
					format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxControlContextDecode, err), http.StatusBadRequest)
					return
				}
			}
			ctrlCtx.LoggerID = loggerID
			ctrlCtx.Service = service
			req := UnmarshalRequest[T](r)
			keys := req.Keys()
			ctrlCtx.Partition = prtMgr.Partition(keys...)

			execCtx = &cc.TxExecutorContext{}
			execCtx.Status = cc.ExecStatusPending
			execCtx.CtrlCtx = ctrlCtx
			execCtx.Input = req

			recorder := mgr.Instrumenter

			ctx = cc.SetTxExecCtx(ctx, execCtx)

			if err = createTxExecutor(
				conn,
				prtMgr,
				clockMgr,
				session,
				execCtx,
				receivers,
			); err != nil {
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxExecutor, err), http.StatusInternalServerError)
				return
			}

			session.Log("Exec Ctx: %v", execCtx)
			recorder.VisitBefore(ctx)
			next.ServeHTTP(w, r.WithContext(ctx))
			recorder.VisitAfter(ctx)
		})
	}
}

func createTxExecutor(
	conn *pgxpool.Pool,
	prtMgr *cc.TxPartitionManager,
	clockMgr *cc.TxClockManager,
	session LoggerSession,
	execCtx *cc.TxExecutorContext,
	receivers []string,
) (err error) {
	var b []byte

	partition := execCtx.CtrlCtx.Partition
	prtMgr.Lock(partition)
	defer prtMgr.Unlock(partition)

	tsMap := map[string]uint64{}
	timestamps := []uint64{}
	for _, receiver := range receivers {
		if ts, ok := tsMap[receiver]; !ok {
			ts := clockMgr.Get(partition, receiver)
			timestamps = append(timestamps, ts+1)
			tsMap[receiver] = ts + 1
		} else {
			timestamps = append(timestamps, ts+1)
			tsMap[receiver]++
		}
	}
	session.Log("ts-map: %v", tsMap)
	execCtx.Receivers = receivers
	execCtx.Timestamps = timestamps

	b, err = json.Marshal(execCtx)
	if err != nil {
		return err
	}

	ctx := context.Background()
	tx, commit, err := database.BeginTx(ctx, conn)
	if err != nil {
		return err
	}
	defer func() {
		err = commit(err)
		session.Log("Tx executor err: %v", err)
		if err == nil {
			// update sender clocks
			for service, newTs := range tsMap {
				clockMgr.Set(partition, service, newTs)
			}
		}
	}()

	insertQuery := `
		INSERT INTO TxExecutor (status, checkpoint)
		VALUES (@status, @checkpoint)
		RETURNING exec_id;
	`
	args := pgx.NamedArgs{
		"status":     cc.ExecStatusPending,
		"checkpoint": b,
	}

	row := tx.QueryRow(ctx, insertQuery, args)
	if err = row.Scan(&execCtx.ExecID); err != nil {
		return err
	}

	batch := &pgx.Batch{}
	count := 0
	// upsert
	timestampQuery := `
		INSERT INTO TxSenderClocks (prt, svc, ts)
		VALUES (@partition, @service, @timestamp)
		ON CONFLICT (prt, svc)
		DO UPDATE SET
			ts = @timestamp;
	`
	for service, newTs := range tsMap {
		args = pgx.NamedArgs{
			"partition": partition,
			"service":   service,
			"timestamp": newTs,
		}
		batch.Queue(timestampQuery, args)
		count++
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	for range count {
		if _, err = results.Exec(); err != nil {
			return err
		}
	}
	return nil
}
