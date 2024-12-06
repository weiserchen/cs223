package v1

import (
	"net/http"
	"txchain/pkg/cc"
	"txchain/pkg/format"
	"txchain/pkg/middleware"
	"txchain/pkg/router"
)

type RequestTestTxUpdateFilter struct {
	FilterType cc.TxFilterType `json:"filter_type"`
	FilterOp   cc.TxFilterOp   `json:"filter_op"`
	Partition  uint64          `json:"partition"`
	Service    string          `json:"string"`
	Attrs      []string        `json:"attrs"`
}

type ResponseTestTxUpdateFilter struct {
}

func HandleTestTxUpdateFilter(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := middleware.UnmarshalRequest[RequestTestTxUpdateFilter](r)
		filterMgr := cfg.TxMgr.FilterMgr
		switch req.FilterType {
		case cc.TxFilterTypeRequest:
			switch req.FilterOp {
			case cc.TxFilterOpAdd:
				filterMgr.AddReqFilter(req.Partition, req.Service, req.Attrs)
			case cc.TxFilterOpRemove:
				filterMgr.RemoveReqFilter(req.Partition, req.Service, req.Attrs)
			case cc.TxFilterOpClear:
				filterMgr.ClearReqFilter(req.Partition, req.Service)
			default:
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrTestTxFilterOp, nil), http.StatusInternalServerError)
				return
			}
		case cc.TxFilterTypeResponse:
			switch req.FilterOp {
			case cc.TxFilterOpAdd:
				filterMgr.AddRespFilter(req.Partition, req.Service, req.Attrs)
			case cc.TxFilterOpRemove:
				filterMgr.RemoveRespFilter(req.Partition, req.Service, req.Attrs)
			case cc.TxFilterOpClear:
				filterMgr.ClearRespFilter(req.Partition, req.Service)
			default:
				format.WriteJsonResponse(w, format.NewErrorResponse(ErrTestTxFilterOp, nil), http.StatusInternalServerError)
				return
			}
		default:
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTestTxFilterType, nil), http.StatusInternalServerError)
			return
		}
		resp := ResponseTestTxUpdateFilter{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestTxAdvanceTimestamp struct {
	Partition uint64 `json:"partition"`
	Service   string `json:"service"`
	Timestamp uint64 `json:"timestamp"`
}

type ResponseTxAdvanceTimestamp struct {
}

func HandleTxAdvanceTimestamp(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := middleware.UnmarshalRequest[RequestTxAdvanceTimestamp](r)
		originMgr := cfg.TxMgr.OriginMgr

		partition, service, timestamp := req.Partition, req.Service, req.Timestamp
		ok := originMgr.Acquire(cc.NewWaitMsg(partition, service, timestamp))
		if ok {
			originMgr.Release(partition, service)
		}

		resp := ResponseTxAdvanceTimestamp{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}
