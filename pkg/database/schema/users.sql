
CREATE TABLE IF NOT EXISTS Users (
  user_id BIGINT GENERATED ALWAYS AS IDENTITY,
  user_name VARCHAR(100) NOT NULL,
  host_events BIGINT[] NOT NULL,
  PRIMARY KEY (user_id),
  Unique (user_name)
);