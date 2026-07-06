# spin

CLI for creating and managing containers in simpler way.

Requires Docker.

## Install

```bash
go install github.com/tymbaca/spin/cmd/spin@latest
```

## Postgres

Start a Postgres container:

```bash
spin up postgres mydb
```

Connect with defaults (`postgres` / `postgres` / `postgres`):

```
postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable
```

Custom port and credentials:

```bash
spin up postgres mydb \
  --port 5433 \
  --user app \
  --password secret \
  --database app
```

Run goose migrations after start:

```bash
spin up postgres mydb --migrations ./db/migrations
```

## Kafka

Start a Kafka container (SASL/PLAIN, user `admin` / password `admin`):

```bash
spin up kafka mykafka
```

Connect from your app:

```
127.0.0.1:9092 SASL_PLAINTEXT PLAIN username=admin password=admin
```

Custom port and credentials:

```bash
spin up kafka mykafka \
  --port 9093 \
  --user myuser \
  --password mypass
```

Create topics inline:

```bash
spin up kafka mykafka --topics orders,payments,events
```

Or from a file (one topic per line, `#` for comments):

```
# topics.txt
orders
payments
events
```

```bash
spin up kafka mykafka --topic-list topics.txt
```

> `--topics` and `--topic-list` cannot be used together.

Start an unconfigured Kafka UI container (available on `:9000` by default):

```bash
spin up kafka-ui my-ui
```

Open http://127.0.0.1:9000

Use a custom Kafka UI port:

```bash
spin up kafka-ui my-ui --port 8080
```

Create Kafka with a configured Kafka UI cluster (separate container named `mykafka-ui`):

```bash
spin up kafka mykafka --ui-port 8080
```

Kafka UI is independent — stopping or removing `mykafka` does not touch `mykafka-ui`:

```bash
spin down mykafka      # kafka stopped, UI still running
spin rm mykafka-ui     # remove UI separately
```

## Redis

Start a Redis container:

```bash
spin up redis myredis
```

Connect with the default port:

```
redis://127.0.0.1:6379
```

Use a custom port:

```bash
spin up redis myredis --port 6380
```

## List containers

```bash
spin ls
```

Show connection strings and URLs:

```bash
spin ls -o wide
```

Example output:

```
NAME      SERVICE    PORT   STATUS   VOLUME
mydb      postgres   5432   running  spin-mydb-data
mykafka   kafka      9092   running  spin-mykafka-data
myredis   redis      6379   running  spin-myredis-data
```

## Start, stop, and remove

Start all stopped spin-managed containers:

```bash
spin up --all
```

Stop a container (keeps data):

```bash
spin down mydb
```

Stop all spin-managed containers:

```bash
spin down --all
```

Remove a container and its volume:

```bash
spin rm mydb
```

Remove all spin-managed containers and their volumes:

```bash
spin rm --all
```

## Port conflicts

If a port is already taken by another spin container, spin asks whether to stop it first:

```
port 5432 is used by spin container "olddb" (running). Spin it down and continue? [y/N]:
```

## Typical workflow

```bash
# start services for a project
spin up postgres api-db --migrations ./migrations
spin up kafka api-kafka --topics orders,payments
spin up kafka-ui api-kafka-ui
spin up redis api-redis

# check what's running
spin ls -o wide

# done for the day
spin down --all

# start everything again
spin up --all

# clean up completely
spin rm --all
```
