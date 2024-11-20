package txclient

import (
	"errors"
	"fmt"
)

type TxType string

var (
	TxTypeCreateEvent TxType = "create-event"
	TxTypeUpdateEvent TxType = "update-event"
	TxTypeDeleteEvent TxType = "delete-event"
	TxTypeJoinEvent   TxType = "join-event"
	TxTypeLeaveEvent  TxType = "leave-event"
	TxTypeDynamic     TxType = "dynamic"
)

var (
	ErrTxTypeUnknown = errors.New("unknown tx type")
)

type TxClient struct {
}

func (client *TxClient) Execute(txType TxType) error {
	switch txType {
	case TxTypeCreateEvent:
	case TxTypeUpdateEvent:
	case TxTypeDeleteEvent:
	case TxTypeJoinEvent:
	case TxTypeLeaveEvent:
	case TxTypeDynamic:
	default:
		return fmt.Errorf("%w: %s", ErrTxTypeUnknown, txType)
	}
	return nil
}
