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

Create topics from a file (one topic per line, `#` for comments):

```
# topics.txt
orders
payments
```

```bash

spin up kafka mykafka --topic-list topics.txt
```

Add Kafka UI (separate container named `mykafka-ui`):

```bash
spin up kafka mykafka --ui-port 8080
```

Open http://127.0.0.1:8080

Kafka UI is independent — stopping or removing `mykafka` does not touch `mykafka-ui`:

```bash
spin down mykafka      # kafka stopped, UI still running
spin rm mykafka-ui     # remove UI separately
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
```

## Stop and remove

Stop a container (keeps data):

```bash
spin down mydb
```

Remove a container and its volume:

```bash
spin rm mydb
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
spin up kafka api-kafka --topic-list topics.txt --ui-port 8080

# check what's running
spin ls -o wide

# done for the day
spin down api-db
spin down api-kafka
spin down api-kafka-ui

# clean up completely
spin rm api-db
spin rm api-kafka
spin rm api-kafka-ui
```
