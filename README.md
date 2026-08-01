# flestudio/pgbackup

A PostgreSQL backup tool for flestudio.

## Features

- Dumps with `pg_dump`, compressed as zstd
- Uploads to S3-compatible storage (e.g. Cloudflare R2)
- Discord notifications (optional)
- Prunes old local backups by retention

## Install

```sh
go install github.com/flestudio/pgbackup/cmd/pgbackup@latest
```

```sh
nix run github:flestudio/pgbackup
```

Needs `pg_dump` on `PATH` (the Nix package bundles it).

## Usage

```sh
pgbackup --s3-bucket my-bucket --s3-endpoint https://<account>.r2.cloudflarestorage.com
```

Options can be passed as flags or env vars, and `.env` is loaded if present.
For the database password, set `PGPASSWORD` or use `.pgpass`.

| Flag                    | Env                   | Default          | Description                                                |
| ----------------------- | --------------------- | ---------------- | ---------------------------------------------------------- |
| `--database`            | `DATABASE`            | `database`       | Database name to back up.                                  |
| `--output-dir`          | `OUTPUT_DIR`          | `./backups`      | Directory to store local backups.                          |
| `--s3-bucket`           | `S3_BUCKET`           | _(required)_     | S3 bucket name.                                            |
| `--s3-region`           | `S3_REGION`           | `auto`           | S3 region.                                                 |
| `--s3-access-key`       | `S3_ACCESS_KEY`       | _(required)_     | S3 access key ID.                                          |
| `--s3-secret-key`       | `S3_SECRET_KEY`       | _(required)_     | S3 secret access key.                                      |
| `--s3-endpoint`         | `S3_ENDPOINT`         | _(required)_     | S3 endpoint URL.                                           |
| `--s3-prefix`           | `S3_PREFIX`           | `backups`        | Key prefix for uploaded objects.                           |
| `--discord-webhook-url` | `DISCORD_WEBHOOK_URL` | _(none)_         | Discord webhook for notifications (optional).              |
| `--log-level`           | `LOG_LEVEL`           | `info`           | Log level (`debug`, `info`, `warn`, `error`).              |
| `--log-format`          | `LOG_FORMAT`          | `text`           | Log format (`text`, `json`).                               |
| `--retention-days`      | `RETENTION_DAYS`      | `7`              | Delete local backups older than this (0 disables pruning). |
| `--compression-level`   | `COMPRESSION_LEVEL`   | `8`              | zstd compression level (1-22).                             |
| `--pg-host`             | `PGHOST`              | _(local socket)_ | PostgreSQL host.                                           |
| `--pg-port`             | `PGPORT`              | _(default)_      | PostgreSQL port.                                           |
| `--pg-user`             | `PGUSER`              | `postgres`       | PostgreSQL user.                                           |

## license

[MIT](./LICENSE)
