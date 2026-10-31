# Auto Myself API

This is a web API which integrates with the Auto Myself application to allow for backup and sharing of vehicles and their maintenance.

## Setup

### Recommended user/database Setup

1. Reconnect to the server, specifying the database and then create the users

```postgresql
CREATE ROLE auto_myself_admin WITH LOGIN PASSWORD 'replace-password' CREATEDB CREATEROLE;
CREATE ROLE auto_myself_api WITH LOGIN PASSWORD 'replace-password';
CREATE DATABASE auto_myself OWNER auto_myself_admin;

\c auto_myself
REVOKE ALL ON SCHEMA public FROM PUBLIC;

GRANT CONNECT ON DATABASE auto_myself TO auto_myself_api;
GRANT USAGE ON SCHEMA public TO auto_myself_api;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO auto_myself_api;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO auto_myself_api;

ALTER DEFAULT PRIVILEGES FOR ROLE auto_myself_admin IN SCHEMA public GRANT SELECT, INSERT, UPDATE ON TABLES TO auto_myself_api;
ALTER DEFAULT PRIVILEGES FOR ROLE auto_myself_admin IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO auto_myself_api;
```

### .env

```
GIN_MODE='debug' # Debug/test use environment variables; release uses Secret Manager
GOOGLE_CLOUD_PROJECT_NUMERIC= # For interacting with Google Cloud APIs, such as Secret Manager
GOOGLE_CLOUD_PROJECT= # For interacting with Google Cloud APIs, such as Cloud Storage
DOMAIN='https://api.example.com' # OAuth redirect base; the callback is /auth/web/callback/google
GOOGLE_OAUTH2_CLIENT_ID=
GOOGLE_OAUTH2_CLIENT_SECRET=
JWT_SIGNING_SECRET= # Generate with `cat /dev/urandom | head -c 32 | sha256sum` or similar
STORAGE_BUCKET= # Optional; defaults to staging.<project>.appspot.com
POSTGRES_DSN='host=127.0.0.1 user=app_user dbname=appdb password=app_user_password port=5432 sslmode=disable' # For local development, works with the compose-database.yml setup
POSTGRES_TEST_DSN='host=127.0.0.1 user=app_user_test dbname=appdb_test password=app_user_test_password port=5433 sslmode=disable' # For tests running locally, works with the compose-database.yml setup
POSTGRES_URL='app_user:app_user_password@localhost:5432/appdb' # For use with the bin/(migrate|seed) scripts
```

`DOMAIN`, `POSTGRES_DSN`, `GOOGLE_OAUTH2_CLIENT_ID`, `GOOGLE_OAUTH2_CLIENT_SECRET`, and
`JWT_SIGNING_SECRET` are required. In `GIN_MODE=debug` or `test` the API reads them from the
environment; in `release` it reads Secret Manager and fails fast if any are missing.

### Running the API

1. Run the database using `docker compose -f compose-database.yml up -d`. You can populate the `.env` file with alternate connection details if you want, ensure that you also update the connection strings for the server and helpers.
2. Run the migrations using `./bin/migrate`
3. Run the seed data using `./bin/seed`
4. Run the API using `go run ./cmd/api`. Using `air-verse/air` or similar will allow for hot reloading during development

### Tests

Tests are integration tests against the Postgres container on port 5433. Each test runs inside a
transaction that is rolled back, and the schema/seed are applied once per test process.

```bash
docker compose -f compose-database.yml up -d
go test -p 1 ./...
```

`-p 1` is required: all test packages share one database. `POSTGRES_TEST_DSN` must be set (it is
loaded from `.env`); otherwise DB-backed tests skip.

### API shape

Every route is served by a Handler → Service → Repository path:
`internal/<domain>/{handlers,services,repositories,models}.go`. Status conventions: 422 malformed
JSON, 400 malformed UUID, 401 unauthenticated, 403 readable-but-forbidden, 404 absent-or-hidden,
201 + `Location` on create, 204 on delete/accept.
