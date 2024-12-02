
-- local timestamp
CREATE TABLE IF NOT EXISTS TxSenderClocks (
  status_id BIGINT GENERATED ALWAYS AS IDENTITY,
  prt BIGINT NOT NULL,
  svc VARCHAR(20) NOT NULL,
  ts BIGINT NOT NULL,
  UNIQUE (svc, prt)
);

CREATE TABLE IF NOT EXISTS TxReceiverClocks (
  status_id BIGINT GENERATED ALWAYS AS IDENTITY,
  prt BIGINT NOT NULL,
  svc VARCHAR(20) NOT NULL,
  ts BIGINT NOT NULL,
  UNIQUE (svc, prt)
);

-- local executor information
CREATE TABLE IF NOT EXISTS TxExecutor (
  exec_id BIGINT GENERATED ALWAYS AS IDENTITY,
  status BIGINT NOT NULL,
  checkpoint JSONB NOT NULL
);

-- result for all partitions
CREATE TABLE IF NOT EXISTS TxResult (
  result_id BIGINT GENERATED ALWAYS AS IDENTITY,
  svc VARCHAR(20) NOT NULL,
  prt BIGINT NOT NULL,
  tx_id BIGINT NOT NULL,
  content BYTEA,
  UNIQUE (svc, prt, tx_id)
);