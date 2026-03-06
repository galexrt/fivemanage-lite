# Fivemanage Lite

Fivemanage Lite is an open-source, lightweight management service designed for gaming communities. It provides essential features for file storage, structured logging, and community organization.

<kbd>
<img width="1790" height="1001" style="border-radius: 15px;" alt="Screenshot 2026-01-18 at 01 43 59" src="https://github.com/user-attachments/assets/6cef883d-3961-495f-b5f6-e3204aac6664" />
</kbd>


## Quick Start

Run the latest version of Fivemanage Lite using Docker:

```bash
docker run -p 8080:8080 \
  -e DSN=postgres://user:pass@host:5432/db \
  -e ADMIN_PASSWORD=your_secure_password \
  -e API_TOKEN_HMAC_SECRET=your_32_byte_secret \
  ghcr.io/fivemanage/lite:0.1.0-beta.18
```

---

## Features

- Multi-tenant organization support.
- File storage with S3-compatible providers (AWS S3, Cloudflare R2, MinIO).
- High-performance structured logging powered by ClickHouse.
- Built-in authentication and session management.
- OpenTelemetry integration for tracing.
- Modern React-based administrative dashboard.

---

## Hosting Options

### Docker Compose
For production environments, using Docker Compose is the most straightforward method to manage the application along with its dependencies (PostgreSQL and ClickHouse). You can find a template in the `deployments/docker-compose.yml` file.

### Kubernetes
Fivemanage Lite is stateless and can be easily deployed on Kubernetes. It is recommended to use a standard Deployment for the application and managed services for the database and storage. Ensure you configure the required environment variables via Secrets or ConfigMaps.

---

## Configuration

The application is configured via environment variables.

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Port the server listens on | `8080` |
| `DSN` | PostgreSQL connection string | - |
| `ADMIN_PASSWORD` | Initial password for the 'admin' user | `password` |
| `API_TOKEN_HMAC_SECRET` | 32-byte secret for signing API tokens | - |
| `S3_PROVIDER` | S3 provider (`s3`, `r2`, or `minio`) | `minio` |
| `AWS_ACCESS_KEY_ID` | S3 access key ID | - |
| `AWS_SECRET_ACCESS_KEY` | S3 secret access key | - |
| `AWS_ENDPOINT` | S3 endpoint URL | - |
| `AWS_BUCKET` | S3 bucket name | - |
| `AWS_REGION` | S3 region | - |
| `CLICKHOUSE_HOST` | ClickHouse host address | `localhost:19000` |
| `CLICKHOUSE_USERNAME` | ClickHouse username | `default` |
| `CLICKHOUSE_PASSWORD` | ClickHouse password | `password` |
| `CLICKHOUSE_DATABASE` | ClickHouse database name | `default` |
| `ENV` | Environment mode (`production` or `dev`) | `dev` |
| `DISABLE_OTEL` | Disable OpenTelemetry integration (true/false) | `false` |

---

## Initial Login

Once the application is running, access the dashboard at `http://localhost:8080`.

- **Username:** `admin`
- **Password:** The value of your `ADMIN_PASSWORD` environment variable.

---

## Development & Contribution

Follow these instructions if you want to contribute to the project or build from source.

### Prerequisites

- **Go**: 1.24 or later
- **Node.js**: 22.x or later
- **pnpm**: 9.x or later
- **Air**: For backend hot-reloading
- **Docker**: For running local infrastructure

### Initial Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/fivemanage/fivemanage-lite.git
   cd fivemanage-lite
   ```

2. **Install Frontend Dependencies:**
   ```bash
   cd web
   pnpm install
   cd ..
   ```

3. **Download Backend Dependencies:**
   ```bash
   go mod download
   ```

4. **Start Development Infrastructure:**
   ```bash
   docker compose -f deployments/docker-compose.yml up -d
   ```

### Running for Development

Run the backend and frontend in separate terminals.

1. **Backend:**
   ```bash
   air
   ```

2. **Frontend:**
   ```bash
   cd web
   pnpm dev
   ```

---

## Contributing

1. **Branching:** Use descriptive branch names (`feature/*`, `fix/*`).
2. **Formatting:** Use `go fmt` and `pnpm lint`.
3. **Commits:** Provide clear, professional messages.
4. **Pull Requests:** Provide a detailed description of changes.

## Project Structure

- `cmd/lite`: Main entry point and CLI configuration.
- `internal/`: Application logic, API handlers, and services.
- `pkg/`: Reusable packages for storage, logging, and caching.
- `web/`: Frontend React application.
- `deployments/`: Docker Compose and deployment configurations.
- `migrate/`: PostgreSQL schema migrations.
- `build/`: Dockerfile and build scripts.

## License

Fivemanage Lite is released under the [MIT License](LICENSE.md).
