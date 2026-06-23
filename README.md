# okfdump

`okfdump` is a Go CLI that connects to a relational database and writes an
[Open Knowledge Format (OKF) v0.1 bundle](https://cloud.google.com/blog/products/data-analytics/how-the-open-knowledge-format-can-improve-data-sharing)
describing its schemas, tables, columns, and foreign-key relationships.

OKF v0.1 is a directory of Markdown files with YAML frontmatter. The bundle this
tool writes is intentionally static and portable: commit it to git, browse it in
an editor, or feed it to an agent as context.

## Install

```sh
go install ./cmd/okfdump
```

## MySQL

```sh
okfdump \
  --driver mysql \
  --host 127.0.0.1 \
  --port 3306 \
  --user app \
  --password "$MYSQL_PASSWORD" \
  --database app_db \
  --out ./okf/app_db
```

Limit the dump to specific tables by repeating `--table` or passing a
comma-separated list. Unqualified names match any schema; qualified names match
`schema.table`.

```sh
okfdump --driver mysql --database app_db --table users --table orders --out ./okf/app_db
okfdump --driver postgres --database app_db --table public.users,sales.orders --out ./okf/app_db
```

## Postgres

```sh
okfdump \
  --driver postgres \
  --host 127.0.0.1 \
  --port 5432 \
  --user app \
  --password "$POSTGRES_PASSWORD" \
  --database app_db \
  --sslmode disable \
  --out ./okf/app_db
```

## DSN

Pass `--dsn` to use a driver-specific connection string directly. When `--dsn`
is set, `--host`, `--port`, `--user`, `--password`, and `--sslmode` are only used
for resource URIs in the generated OKF files unless an SSH tunnel is enabled.

```sh
okfdump --driver mysql --dsn 'user:pass@tcp(localhost:3306)/app_db' --database app_db --out ./okf/app_db
```

## SSH Tunnel

When `--ssh-host` is set, the CLI opens a local TCP listener on `127.0.0.1` and
forwards database traffic through the SSH server to `--host:--port`.
Host keys are checked against `~/.ssh/known_hosts` by default. For disposable
local testing only, pass `--ssh-insecure-ignore-host-key`.

```sh
okfdump \
  --driver mysql \
  --host private-db.internal \
  --port 3306 \
  --user app \
  --password "$MYSQL_PASSWORD" \
  --database app_db \
  --out ./okf/app_db \
  --ssh-host bastion.example.com \
  --ssh-user deploy \
  --ssh-key ~/.ssh/id_ed25519
```

## Output Shape

```text
bundle/
├── index.md
├── log.md
├── databases/
│   ├── index.md
│   └── app_db.md
├── schemas/
│   ├── index.md
│   └── public.md
└── tables/
    ├── index.md
    └── public/
        ├── index.md
        └── users.md
```

Each non-reserved concept file includes OKF frontmatter with a required `type`
field plus recommended `title`, `description`, `resource`, `tags`, and
`timestamp` fields.
