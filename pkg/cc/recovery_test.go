package cc

import (
	"context"
	"testing"
	"time"
	"txchain/pkg/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

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

	partitions := uint64(10)
	services := []string{"service-a", "service-b", "service-c"}

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
			_, err = results.Exec()
			require.NoError(t, err)
		}
	}

	sendClockMgr := NewTxClockManager(partitions)
	recvClockMgr := NewTxClockManager(partitions)
	execMgr := NewTxExecutorManager(ExponentialBackoffRetry(100 * time.Millisecond))
	recoveryMgr := NewTxRecoveryManager(conn, sendClockMgr, recvClockMgr, execMgr)
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

}
