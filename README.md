# Cinna

A Telegram assistant written in Go.

## Requirements

- Go 1.26+
- Docker
- `sqlc`
- A Telegram bot token

## Setup

Create and configure the local environment:

```bash
cp .env.example .env

set -a
source .env
set +a
```

Start PostgreSQL:

```bash
docker compose up -d postgres
```

On first startup, PostgreSQL creates the database named by `POSTGRES_DB`.
Validate the server and database from the CLI:

```bash
docker compose exec postgres \
  pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"

docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT current_database(), current_user;"
```

Apply the schema:

```bash
docker compose exec -T postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" < db/schema.sql
```

Validate that the table was created:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "\d allowed_users"
```

Generate Go code from the SQL queries:

```bash
sqlc generate
```

Run Cinna:

```bash
go run ./cmd/cinna
```

## SQL Commands

Open a PostgreSQL shell:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
```

List allowed users:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT * FROM allowed_users ORDER BY updated_at;"
```

Add or reactivate an allowed user:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "INSERT INTO allowed_users (telegram_user_id, role)
      VALUES (12345678, 'admin')
      ON CONFLICT (telegram_user_id) DO UPDATE
      SET role = EXCLUDED.role, is_active = TRUE, updated_at = NOW();"
```

Regenerate code after changing `db/schema.sql` or files under `db/queries/`:

```bash
sqlc generate
```

Stop PostgreSQL:

```bash
docker compose down
```

Delete PostgreSQL data:

```bash
docker compose down -v
```

## Checks

```bash
go test ./...
go vet ./...
```

## Project Structure

```text
cinna/
├── cmd/
│   └── cinna/
├── db/
│   ├── queries/
│   └── schema.sql
├── internal/
│   ├── app/
│   │   └── telegram/
│   └── postygres/
│       └── sqlc/
├── prompts/
│   ├── core/
│   └── shopping/
├── compose.yaml
├── go.mod
└── sqlc.yaml
```
