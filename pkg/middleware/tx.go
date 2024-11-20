package middleware

import (
	"context"
	"errors"
	"sync"
)

type TxType int

var (
	TxTypeRead  TxType = 0
	TxTypeWrite TxType = 1
)

var (
	ErrUnknownTxType = errors.New("unknown tx type")
	ErrTryReadLock   = errors.New("failed to get read lock")
	ErrTryWriteLock  = errors.New("failed to get write lock")
)

type TxGuard interface {
	AcquireGlobal(ctx context.Context, txType TxType) error
	ReleaseGlobal(txType TxType)
	Acquire(ctx context.Context, key any) error
	Release(key any)
}

type MutexTxGuard struct {
	atom      sync.RWMutex
	partition map[any]sync.Mutex
}

func (guard *MutexTxGuard) AcquireGlobal(_ context.Context, txType TxType) error {
	switch txType {
	case TxTypeRead:
		if guard.atom.TryRLock() {
			return nil
		} else {
			return ErrTryReadLock
		}
	case TxTypeWrite:
		if guard.atom.TryLock() {
			return nil
		} else {
			return ErrTryWriteLock
		}
	default:
		return ErrUnknownTxType
	}
}

func (guard *MutexTxGuard) ReleaseGlobal(txType TxType) {
	switch txType {
	case TxTypeRead:
		guard.atom.RUnlock()
	case TxTypeWrite:
		guard.atom.Unlock()
	default:
		return
	}
}

type ChannelTxGuard struct {
}

type RWLocker interface {
	RLock()
	RUnlock()
	Lock()
	Unlock()
}

var _ RWLocker = (*MutexRWLock)(nil)

type MutexRWLock struct {
	sync.RWMutex
}

type ChannelRWLock struct {
}
