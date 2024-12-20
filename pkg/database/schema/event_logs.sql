
CREATE TABLE IF NOT EXISTS EventLogs (
  log_id BIGINT GENERATED ALWAYS AS IDENTITY,
  event_id BIGINT,
  user_id BIGINT,
  event_type VARCHAR(20) NOT NULL,
  content VARCHAR(1000) NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (log_id)
);