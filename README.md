# Configuration Reference

This application uses `config/config.yaml` as the primary configuration source. Values in the YAML file are loaded at startup and can be overridden with environment variables.

## Configuration file structure

```yaml
server:
  port: ":8080"

database:
  dsn: "root:admin@jPASSWORDMO!@tcp(127.0.0.1:3306)/gin_api?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
```

### server
- `port`: The address the Gin server binds to.
- Defaults to `:8080` when not set in config or environment.
- Can be overridden by the `PORT` environment variable.

### database
- `dsn`: Database connection string.
- Recommended to use environment overrides for passwords and production secrets.

### redis
- `addr`: Redis host and port.
- `password`: Redis authentication password.
- `db`: Redis database index.

## Environment variable overrides

The application supports the following environment variables:

- `PORT`: Overrides `server.port`.
- `DATABASE_DSN` or `DB_DSN`: Overrides `database.dsn`.
- `REDIS_ADDR`: Overrides `redis.addr`.
- `REDIS_PASSWORD`: Overrides `redis.password`.
- `REDIS_DB`: Overrides `redis.db`.

## Defaults and behavior

- `server.port`: default `:8080`
- `redis.addr`: default `127.0.0.1:6379`

If `PORT` contains a colon (`:`), it is used verbatim; otherwise the application prepends `:` automatically.

## Usage

1. Update `config/config.yaml` for local development.
2. Use environment variables in deployment or CI to avoid committing secrets.
3. Start the app with `go run .` and verify the server binds to the configured port.
