# Gin Concurrency

This project runs a Gin API with MySQL and Redis through Docker Compose.

For configuration details, see [config/README.md](config/README.md).

## Docker Commands

Use Docker Compose v2 commands from the repository root.

### Build and start

```sh
# Build the app image, start app/mysql/redis, and stream logs in the terminal.
docker compose up --build

# Build the app image and start all services in the background.
docker compose up -d --build

# Start existing containers in the background without rebuilding images.
docker compose up -d
```

### Service status and traceability

```sh
# Show each service, container state, and published ports.
docker compose ps

# Show the fully rendered compose file after environment variable interpolation.
docker compose config

# Follow all service logs with timestamps for request and startup tracing.
docker compose logs -f --timestamps --tail=100

# Follow only the Gin API logs.
docker compose logs -f --timestamps --tail=100 app

# Follow only Redis logs.
docker compose logs -f --timestamps --tail=100 redis

# Follow only MySQL logs.
docker compose logs -f --timestamps --tail=100 mysql

# Show recent logs from the last 10 minutes.
docker compose logs --timestamps --since=10m

# Show live Docker events for this Compose project.
docker compose events

# Show running processes inside each service container.
docker compose top
```

### Shell access and inspection

```sh
# Open a shell inside the app container.
docker compose exec app sh

# Print environment variables inside the app container.
docker compose exec app env

# Open an interactive Redis CLI session.
docker compose exec redis redis-cli

# Open an interactive MySQL session using the container environment.
docker compose exec mysql sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE"'

# Show the container ID for the app service.
docker compose ps -q app

# Show the container ID for the Redis service.
docker compose ps -q redis
```

### Rebuild, stop, and cleanup

```sh
# Rebuild only the app image.
docker compose build app

# Rebuild the app image without using Docker layer cache.
docker compose build --no-cache app

# Restart the app service after a rebuild or config change.
docker compose restart app

# Stop containers but keep them available for another start.
docker compose stop

# Stop and remove containers and the Compose network. Named volumes are kept.
docker compose down

# DANGER: Stop containers and remove MySQL/Redis named volumes.
docker compose down -v

# Remove stopped service containers.
docker compose rm -f
```

## Redis Commands

The Compose service uses Redis `7.4-alpine` with append-only persistence enabled.
The app currently validates Redis connectivity at startup.

### Health checks

```sh
# Confirm Redis accepts commands.
docker compose exec redis redis-cli ping

# Show Redis server metadata.
docker compose exec redis redis-cli info server

# Show Redis memory usage.
docker compose exec redis redis-cli info memory

# Show Redis persistence state, including AOF/RDB details.
docker compose exec redis redis-cli info persistence

# Confirm append-only persistence is enabled.
docker compose exec redis redis-cli config get appendonly

# Show the selected database key count.
docker compose exec redis redis-cli dbsize
```

### Trace Redis activity

```sh
# Trace every Redis command received by the server. Stop with Ctrl+C.
docker compose exec redis redis-cli monitor

# Show connected Redis clients.
docker compose exec redis redis-cli client list

# Show the latest slow commands.
docker compose exec redis redis-cli slowlog get 10

# Reset the slow command log after capturing it.
docker compose exec redis redis-cli slowlog reset
```

### Inspect keys

```sh
# Scan keys safely in batches.
docker compose exec redis redis-cli --scan

# Scan keys by pattern.
docker compose exec redis redis-cli --scan --pattern "user:*"

# Show the Redis data type for a key.
docker compose exec redis redis-cli type "<key>"

# Show the remaining time to live for a key in seconds.
docker compose exec redis redis-cli ttl "<key>"

# Read a string key.
docker compose exec redis redis-cli get "<key>"

# Read a hash key.
docker compose exec redis redis-cli hgetall "<key>"

# Read a list key.
docker compose exec redis redis-cli lrange "<key>" 0 -1

# Read a set key.
docker compose exec redis redis-cli smembers "<key>"

# Read a sorted set key with scores.
docker compose exec redis redis-cli zrange "<key>" 0 -1 withscores
```

### Change or clear Redis data

```sh
# Delete one key.
docker compose exec redis redis-cli del "<key>"

# Set a test key with a 60 second expiration.
docker compose exec redis redis-cli setex trace:test 60 "ok"

# Verify the test key value.
docker compose exec redis redis-cli get trace:test

# DANGER: Remove all keys from the selected Redis database.
docker compose exec redis redis-cli flushdb

# DANGER: Remove all keys from every Redis database.
docker compose exec redis redis-cli flushall
```

### Persistence and volume checks

```sh
# Ask Redis to create a background snapshot.
docker compose exec redis redis-cli bgsave

# List Redis persistence files inside the container.
docker compose exec redis ls -lah /data

# List Docker volumes so you can find this project's mysql_data/redis_data volumes.
docker volume ls

# Copy Redis persistence files from the container to a local backup folder.
docker compose cp redis:/data ./redis-data-backup
```

## API Smoke Commands

Use these after `docker compose up -d --build`.

```sh
# Seed a small number of users for a quick test.
curl -X POST http://localhost:${APP_PORT:-8080}/seed/users \
  -H "Content-Type: application/json" \
  -d '{"total_rows":1000,"batch_size":100,"game_id":1,"workers":4}'

# Fetch a small batch of users.
curl "http://localhost:${APP_PORT:-8080}/fetch/users?total_rows=100&batch_size=50&workers=4"

# Stream user batches as newline-delimited JSON.
curl -N "http://localhost:${APP_PORT:-8080}/fetch/users/channel?total_rows=100&batch_size=50&workers=4"
```
