
CREATE TABLE IF NOT EXISTS EventLogs (
  log_id BIGINT GENERATED ALWAYS AS IDENTITY,
  event_id BIGINT,
  user_id BIGINT,
  event_type VARCHAR(20) NOT NULL,
  update VARCHAR(200) NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (log_id)
);

CREATE OR REPLACE FUNCTION updated_at_trigger()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_event_log_updated_at
    BEFORE UPDATE
    ON
        EventLogs
    FOR EACH ROW
EXECUTE PROCEDURE updated_at_trigger();

