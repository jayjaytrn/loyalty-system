# Configuration Service

This service provides configuration for the application, supporting the loading of values from environment variables and command-line arguments.

## Installation and Setup

### Environment Variables

The service uses the following environment variables for configuration:

- **RUN_ADDRESS** (required) — the address where the application will run (e.g., `localhost:8080`).
- **DATABASE_URI** (required) — the PostgreSQL connection string (e.g., `postgres://admin:admin@localhost:5432/test`).
- **ACCRUAL_SYSTEM_ADDRESS** (required) — the address of the accrual system (e.g., `http://localhost:9000`).
- **ACCRUAL_REQUEST_TIMEOUT** — the timeout for accrual system requests, in seconds (default: `5s`).
- **RECOVERY_INTERVAL** — the interval between recovery attempts, in seconds (default: `5s`).
- **RECOVERY_RETRY_INTERVAL** — the duration for retrying recovery attempts, in minutes (default: `60m`).
- **WORKER_COUNT** — the number of workers for processing tasks (default: `10`).

### Command-Line Arguments

You can also specify configuration parameters via command-line arguments. The following arguments are available with their default values:

- `-a` or `--RunAddress` — the address to run the service on (default: `localhost:8080`).
- `-d` or `--DatabaseURI` — the database URI (default: `postgres://admin:admin@localhost:5432/test`).
- `-r` or `--AccrualSystemAddress` — the address of the accrual system (default: `test`).
- `-t` or `--AccrualRequestTimeoutSeconds` — the timeout for accrual system requests (default: `5`).
- `-i` or `--RecoveryIntervalSeconds` — the recovery interval, in seconds (default: `5`).
- `-m` or `--RecoveryRetryDurationMinutes` — the retry duration for recovery attempts, in minutes (default: `60`).
- `-w` or `--WorkerCount` — the number of workers (default: `10`).

### Example Usage

To start the service with custom values:

```bash
RUN_ADDRESS="localhost:8080" DATABASE_URI="postgres://user:password@localhost:5432/db" ACCRUAL_SYSTEM_ADDRESS="http://localhost:9000" ./your-app
