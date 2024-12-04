package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"txchain/pkg/cc"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMiddlewareTxGuard           = errors.New("failed to acuquire tx lock")
	ErrMiddlewareTxExecutor        = errors.New("failed to create tx executor")
	ErrMiddlewareTxMiddsingCtrlCtx = errors.New("missing control context")
)

func TxTraceContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceCtx := format.NewTraceContext()
		ctx := format.SetTraceContext(r.Context(), traceCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TxStageContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoded := r.Header.Get(headerTxStageContext)
		// No tx context
		if encoded == "" {
			next.ServeHTTP(w, r)
			return
		}

		stageCtx, err := cc.DecodeTxStageContext(encoded)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxStageContextDecode, err), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyTxStageContext, stageCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TxControlContext(service string) Middlerware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			encoded := r.Header.Get(headerTxControlContext)
			// No tx context
			if encoded == "" {
				next.ServeHTTP(w, r)
				return
			}

			ctrlCtx, err := cc.DecodeTxControlContext(encoded)
			if err != nil {
				format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxControlContextDecode, err), http.StatusBadRequest)
				return
			}
			ctrlCtx.Service = service

			ctx := context.WithValue(r.Context(), contextKeyTxControlContext, ctrlCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TxExecutorContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoded := r.Header.Get(headerTxExecutorContext)
		// No tx context
		if encoded == "" {
			next.ServeHTTP(w, r)
			return
		}

		execCtx, err := cc.DecodeTxExecutorContext(encoded)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxExecutorContextDecode, err), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyTxExecutorContext, execCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TxPartition[T cc.Partition](mgr *cc.TxPartitionManager) Middlerware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctrlCtx, ok := MarshalTxControlContext(r)
			// No tx control
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			req := MarshalRequest[T](r)
			keys := req.Keys()

			partition := mgr.Partition(keys...)
			ctrlCtx.Partition = partition
			mgr.Lock(partition)
			defer mgr.Unlock(partition)

			ctx := context.WithValue(r.Context(), contextKeyTxControlContext, ctrlCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type ResponseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWrapper(w http.ResponseWriter) *ResponseWrapper {
	return &ResponseWrapper{}
}

func (rw *ResponseWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func TxOriginOrdering(mgr *cc.TxOriginManager) Middlerware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			stageCtx, ok := MarshalTxStageContext(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			partition, service, timestamp := stageCtx.Partition, stageCtx.Service, stageCtx.Timestamp
			mgr.Acquire(cc.NewWaitMsg(partition, service, timestamp))

			// TODO: handle request errors
			next.ServeHTTP(w, r)

			mgr.Release(partition, service)
		})
	}
}

func TxFilter(mgr *cc.TxFilterManager) Middlerware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			stageCtx, ok := MarshalTxStageContext(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			partition := stageCtx.Partition
			service := stageCtx.Service
			attrs := stageCtx.Attrs
			dropReq := mgr.DropReq(partition, service, attrs)
			dropResp := mgr.DropResp(partition, service, attrs)
			if dropReq {
				format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxRequestDropped, nil), http.StatusServiceUnavailable)
				return
			}

			writer := w
			if dropResp {
				writer = httptest.NewRecorder()
			}
			next.ServeHTTP(writer, r)

			if dropResp {
				format.WriteJsonResponse(w, format.NewErrorResponse(cc.ErrTxResponseDropped, nil), http.StatusServiceUnavailable)
				return
			}
		})
	}
}

func TxExecutor[T cc.Partition](
	conn *pgxpool.Pool,
	mgr *cc.TxClockManager,
	receivers []string,
) Middlerware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var execCtx *cc.TxExecutorContext
			var ctrlCtx *cc.TxControlContext
			var ok bool
			var err error
			var tx pgx.Tx

			_, ok = MarshalTxExecutorContext(r)
			if ok {
				next.ServeHTTP(w, r)
				return
			}
			execCtx = &cc.TxExecutorContext{}
			execCtx.Status = cc.ExecStatusPending

			ctrlCtx, ok = MarshalTxControlContext(r)
			if !ok {
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxMiddsingCtrlCtx, err), http.StatusInternalServerError)
				return
			}
			execCtx.CtrlCtx = ctrlCtx

			partition := ctrlCtx.Partition
			// Increment timestamps from sender sides
			tsMap := map[string]uint64{}
			timestamps := []uint64{}
			for _, receiver := range receivers {
				if ts, ok := tsMap[receiver]; !ok {
					ts := mgr.Get(partition, receiver)
					timestamps = append(timestamps, ts)
					tsMap[receiver] = ts
				} else {
					timestamps = append(timestamps, ts+1)
					tsMap[receiver]++
				}
			}
			execCtx.Receivers = receivers
			execCtx.Timestamps = timestamps

			b, err := json.Marshal(execCtx)
			if err != nil {
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxExecutor, err), http.StatusInternalServerError)
				return
			}

			ctx := r.Context()
			tx, err = conn.Begin(ctx)
			if err != nil {
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxExecutor, err), http.StatusInternalServerError)
				return
			}

			insertQuery := `
				INSERT INTO TxExecutor (checkpoint)
				VALUES ($1)
				RETURNING exec_id;
			`

			row := tx.QueryRow(ctx, insertQuery, "{}")
			if err = row.Scan(&execCtx.ExecID); err != nil {
				_ = tx.Rollback(ctx)
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxExecutor, err), http.StatusInternalServerError)
				return
			}

			batch := &pgx.Batch{}
			count := 0
			updateQuery := `
				UPDATE TxExecutor (checkpoint)
				VALUES ($2);
				WHERE exec_id = $1; 
			`
			batch.Queue(updateQuery, execCtx.ExecID, b)
			count++

			for service, ts := range tsMap {
				newTs := ts + 1
				// upsert
				timestampQuery := `
					INSERT INTO TxSenderClocks (prt, svc, ts)
					VALUES ($1, $2, $3)
					ON CONFLICT (prt, svc)
					DO UPDATE SET
						ts = $3
				`
				batch.Queue(timestampQuery, partition, service, newTs)
				count++
			}

			results := tx.SendBatch(ctx, batch)
			defer results.Close()

			for range count {
				if _, err = results.Exec(); err != nil {
					_ = tx.Rollback(ctx)
					format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxExecutor, err), http.StatusInternalServerError)
					return
				}
			}

			if err = tx.Commit(ctx); err != nil {
				_ = tx.Rollback(ctx)
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrMiddlewareTxExecutor, err), http.StatusInternalServerError)
				return
			}

			// update sender clocks
			for service, ts := range tsMap {
				newTs := ts + 1
				mgr.Set(partition, service, newTs)
			}

			ctx = context.WithValue(ctx, contextKeyTxExecutorContext, execCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
