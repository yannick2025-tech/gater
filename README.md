# NTS-Gater

EV charging station communication protocol conformance testing gateway. Acts as the "platform side" to communicate with charging stations via TCP, drive automated test scenarios, record message traces, and generate PDF test reports.

## Architecture

```
Charging Station ──TCP──→ [TCP Server] ──→ [Dispatcher] ──→ [Session Manager]
                                                      │
                                                      ├── [Scenario Engine] (test orchestration)
                                                      ├── [Recorder] (message logging)
                                                      ├── [Protocol Codec] (frame encoding/decoding + AES-CBC encryption)
                                                      └── [Report Service] (PDF generation)
                                                                │
Web Frontend (Vue SPA) ──HTTP──→ [Gin Router /api/*] ────────────┘
```

## Ports

| Port | Service | Description |
|------|---------|-------------|
| 8888 | TCP | Charging station connections (protocol communication) |
| 9090 | API | Internal HTTP API (`/api/*` only, not exposed externally) |
| 8080 | Web | Public-facing static files, SPA, and Swagger UI |
| 3000 | Vite Dev | Frontend dev server with API proxy to 9090 (development only) |

## Prerequisites

- **Go** 1.25+ ([go.dev/dl](https://go.dev/dl/))
- **Node.js** 18+ ([nodejs.org](https://nodejs.org/))
- **MySQL** 5.7+ (for test report persistence)

## Quick Start

```bash
# Clone the repository
git clone https://github.com/yannick2025-tech/nts-gater.git
cd nts-gater

# Install Go dependencies
go mod download

# Install frontend dependencies (MUST be inside the web/ directory)
cd web
npm install
cd ..

# Configure the application
cp configs/config.yaml configs/config.local.yaml
# Edit config as needed (database DSN, ports, protocol settings, etc.)

# Start development environment (backend + frontend dev server)
make dev
```

After `make dev`, the following services are available:

- Frontend: [http://localhost:3000](http://localhost:3000)
- API (internal): [http://localhost:9090](http://localhost:9090)
- Web (production static): [http://localhost:8080](http://localhost:8080)
- Swagger UI: [http://localhost:8080/swagger](http://localhost:8080/swagger)

## Production Build

```bash
# Build frontend assets
make web

# Build Go binary
make build

# Run
./build/nts-gater
```

The production deployment uses ports 8888 (TCP), 9090 (API), and 8080 (Web). The Vite dev server (3000) is not used in production.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/sessions` | List all TCP sessions (active + historical) |
| GET | `/api/device/status?gunNumber=` | Query device online/auth status |
| POST | `/api/device/connect` | Connect device from web UI |
| POST | `/api/device/disconnect` | Disconnect device (async cleanup) |
| POST | `/api/test/start` | Start a test scenario |
| GET | `/api/test/status/:sessionId` | Query test progress |
| GET | `/api/test/results` | Paginated test results |
| GET | `/api/test/detail/:sessionId` | Test detail with func code stats |
| GET | `/api/test/messages/:sessionId` | Message archive for a session |
| POST | `/api/test/decode` | Decode a hex message |
| POST | `/api/test/export` | Export test report as PDF |
| GET | `/api/test/download?path=` | Download PDF file |
| POST | `/api/test/config` | Push configuration to charging station |

## Test Scenarios

| Scenario | Key | Description |
|----------|-----|-------------|
| Basic Charging | `basic_charging` | Full charging flow: 0x03 start → 0x04 confirm → 0x06 charging data → 0x08 stop → 0x05 stop confirm |
| SFTP Upgrade | `sftp_upgrade` | Remote firmware upgrade via SFTP |
| Config Download | `config_download` | Platform pushes configuration parameters (0xC2, etc.) |

## Project Structure

```
.
├── cmd/server/            # Application entry point
├── configs/               # Configuration files (YAML)
├── docs/api/              # OpenAPI specification
├── internal/
│   ├── api/               # HTTP routes and handlers
│   ├── config/            # Configuration loading
│   ├── database/          # Database initialization (MySQL/GORM)
│   ├── dispatcher/        # Message dispatcher
│   ├── errors/            # Error code definitions
│   ├── generator/         # Order number / ID generators
│   ├── handlers/          # Auth, charging, and misc handlers
│   ├── model/             # Database models (TestReport, FuncCodeStat, MessageArchive)
│   ├── protocol/
│   │   ├── codec/         # Frame encoding, checksum, scanner
│   │   ├── crypto/        # AES-CBC encryption
│   │   ├── standard/      # Standard protocol definitions & enums
│   │   │   └── msg/       # Message structs per function code
│   │   └── types/         # Protocol/message interfaces
│   ├── recorder/          # Session message recorder
│   ├── report/            # PDF report generation & DB queries
│   ├── scenario/          # Test scenario engine & implementations
│   ├── server/            # TCP server
│   ├── session/           # Session manager (auth, key state)
│   └── validator/         # Frame validation
├── web/                   # Vue 3 frontend (SPA)
│   ├── src/
│   │   ├── views/         # Page components
│   │   ├── router/        # Vue Router config
│   │   └── ...
│   └── package.json
├── Makefile
└── go.mod
```

## Configuration

All settings are in `configs/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8888              # TCP port for charging station connections
  http_port: 9090         # API port (internal, /api/* only)
  web_port: 8080          # Web port (public, static files + SPA)
  heartbeat_timeout: 3m

database:
  driver: "mysql"
  dsn: "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"

protocol:
  name: "standard"
  version: "06"
  fixed_key: "..."         # AES fixed key for protocol encryption
```

## Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile Go binary to `build/nts-gater` |
| `make run` | Run the backend directly |
| `make dev` | Start backend + Vite dev server concurrently |
| `make web` | Build frontend (`npm install && npm run build`) |
| `make web-dev` | Start Vite dev server only |
| `make test` | Run Go unit tests |
| `make test-coverage` | Run tests with coverage report |
| `make tidy` | Tidy Go modules |
| `make fmt` | Format Go code |
| `make lint` | Run golangci-lint |
| `make clean` | Remove build artifacts and frontend dist |

## License

Proprietary. All rights reserved.
