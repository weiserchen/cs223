# cs223

## Install

```bash
# macos
brew install libpq
brew link --force libpq

psql postgres://postgres:postgres@localhost:15432/calendar
```

## Usage
```bash
# build service
make build

# run database (will stop automatically)
make run
```

## Database
```sql
-- commands;
\x -- turn on extended display
\dt; -- list all databases

-- drop tables
DROP TABLE Users CASCADE;
DROP TABLE Events CASCADE;
DROP TABLE EventLogs CASCADE;

-- truncate tables
TRUNCATE TABLE Users CASCADE;
TRUNCATE TABLE Events CASCADE;
TRUNCATE TABLE EventLogs CASCADE;
```


## Troubleshooting

### Podman VSCode config
```json
{
    "go.testEnvVars": {
        "DOCKER_HOST": "unix:///var/run/docker.sock",
        "TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE": "/var/run/docker.sock",
        "TESTCONTAINERS_RYUK_DISABLED": "true",
        "TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED": "true",
        "POSTGRES_PASSWORD": "postgres",
        "POSTGRES_USER": "postgres",
        "POSTGRES_DB": "test"
    },
    "go.testFlags": ["-count=1"]
}
```

## References
- [project paper](https://dl.acm.org/doi/10.1145/2517349.2522729)
- https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/
