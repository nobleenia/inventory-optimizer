# Inventory Optimizer

A lightweight inventory decision engine for e-commerce sellers. It analyses historical sales data, computes optimal reorder policies, and simulates future inventory performance using Monte-Carlo methods — available as a CLI tool, a browser-based web application, and a REST API with JWT authentication.

**Version:** 0.3.0  
**Language:** Go 1.21+  
**Author:** Noble Eluwah

---

## What It Does

For each SKU in your catalogue the engine produces:

| Output | Description |
|---|---|
| **Reorder Point** | Inventory level at which to place a new order |
| **Order Quantity (EOQ)** | Optimal number of units per order |
| **Safety Stock** | Buffer inventory to guard against demand variability |
| **Expected Stockouts** | Estimated stockout events per year |
| **Average Inventory** | Mean units on hand across the simulation |
| **Annual Cost** | Estimated holding + ordering costs |

---

## Quick Start

### Option A — Command Line

```bash
# Build
go build -o inventory-optimizer ./cmd/

# Run with sample data
./inventory-optimizer \
  -sales  data/sales_history.csv \
  -params data/sku_parameters.csv \
  -output output/report.csv
```

Results are printed to the terminal and optionally exported as CSV.

### Option B — Web Interface

```bash
# Build
go build -o inventory-optimizer ./cmd/

# Launch the web server
./inventory-optimizer -web

# Open your browser at http://localhost:8080
```

Upload your CSV files through the browser, review per-SKU results with plain-English recommendations, and download a CSV report — no terminal knowledge required.

### Option C — REST API

```bash
# Start PostgreSQL (Docker)
docker-compose up -d

# Build & launch the API server
go build -o inventory-optimizer ./cmd/
./inventory-optimizer -api

# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"seller@example.com","password":"securepass123"}'

# Login (copy the access_token from the response)
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"seller@example.com","password":"securepass123"}'

# Run analysis
curl -X POST http://localhost:8080/api/analyze \
  -H "Authorization: Bearer <TOKEN>" \
  -F "sales_file=@data/sales_history.csv" \
  -F "params_file=@data/sku_parameters.csv" \
  -F "title=Q1 Analysis"

# List saved reports
curl http://localhost:8080/api/reports \
  -H "Authorization: Bearer <TOKEN>"
```

Full API documentation is in [docs/openapi.yaml](docs/openapi.yaml).

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-web` | `false` | Launch the web interface instead of CLI mode |
| `-api` | `false` | Launch the REST API server (requires PostgreSQL) |
| `-port` | `:8080` | Port for the web/API server |
| `-sales` | `data/sales_history.csv` | Path to weekly sales history CSV |
| `-params` | `data/sku_parameters.csv` | Path to SKU parameters CSV |
| `-output` | *(none)* | Path for CSV export (omit to skip) |
| `-version` | — | Print version and exit |

### Environment Variables (API mode)

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://inventory:inventory@localhost:5433/inventory?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | *(dev default)* | HMAC-SHA256 signing key for JWT tokens |

---

## Input Files

### `sales_history.csv`

Weekly sales by SKU.

```csv
sku,week,units_sold
SKU001,2024-01-01,12
SKU001,2024-01-08,15
```

- One row per SKU per week
- Date format: `YYYY-MM-DD`
- Missing weeks are treated as zero demand

### `sku_parameters.csv`

Product and cost parameters.

```csv
sku,current_inventory,lead_time_days,unit_cost,order_cost,holding_cost_rate
SKU001,120,21,8.50,40.00,0.25
```

| Column | Meaning |
|---|---|
| `current_inventory` | Units currently on hand |
| `lead_time_days` | Days between order placement and delivery |
| `unit_cost` | Purchase cost per unit (€) |
| `order_cost` | Fixed cost per order (€) |
| `holding_cost_rate` | Annual carrying cost as a fraction (0.25 = 25%) |

---

## Project Structure

```
inventory-optimizer/
├── cmd/
│   └── main.go                  # Tri-mode entry point (CLI / web / API)
├── internal/
│   ├── models/
│   │   └── sku.go               # Core data types shared across packages
│   ├── parser/
│   │   ├── csv_reader.go        # CSV ingestion & validation
│   │   └── csv_reader_test.go   # 11 tests
│   ├── demand/
│   │   ├── statistics.go        # Demand statistical analysis
│   │   └── statistics_test.go   # 7 tests
│   ├── inventory/
│   │   ├── eoq.go               # Economic Order Quantity
│   │   ├── safety_stock.go      # Safety stock (Z-score approach)
│   │   ├── reorder_point.go     # Reorder point
│   │   ├── policy.go            # Unified policy computation
│   │   └── inventory_test.go    # 10 tests
│   ├── simulation/
│   │   ├── monte_carlo.go       # Monte-Carlo inventory simulation
│   │   └── monte_carlo_test.go  # 6 tests
│   ├── reporting/
│   │   ├── results.go           # CLI display & CSV export
│   │   └── results_test.go      # 5 tests
│   ├── engine/
│   │   └── engine.go            # High-level pipeline orchestrator
│   ├── auth/
│   │   ├── jwt.go               # JWT token creation & validation, bcrypt
│   │   └── jwt_test.go          # 12 tests
│   ├── store/
│   │   ├── postgres.go          # Connection pool, migrations
│   │   ├── users.go             # User CRUD
│   │   └── reports.go           # Report CRUD (JSONB storage)
│   ├── api/
│   │   ├── router.go            # REST API routes & server
│   │   ├── handlers.go          # Auth, analyze, reports, profile handlers
│   │   ├── middleware.go        # JWT auth, CORS, request logging
│   │   ├── ratelimit.go         # In-memory token bucket rate limiter
│   │   ├── response.go          # JSON envelope helpers
│   │   └── api_test.go          # 9 tests
│   └── web/
│       ├── server.go            # HTML web server, routes & handlers
│       ├── templates/           # Embedded HTML templates
│       └── static/              # CSS & JS
├── docs/
│   ├── WORKING_DOC.md           # Project specification
│   └── openapi.yaml             # OpenAPI 3.0 spec for the REST API
├── docker-compose.yml           # PostgreSQL 16 container
├── .env.example                 # Environment variable template
└── data/
    ├── sales_history.csv        # Sample sales data (3 SKUs × 52 weeks)
    └── sku_parameters.csv       # Sample SKU config
```

Each package has a **single responsibility** and communicates only through types defined in `models/`. The `engine` package orchestrates the full pipeline so that CLI, web, and API modes share one code path.

---

## Tests

```bash
go test ./... -v
```

57 unit tests across 7 packages covering parsing, statistics, inventory calculations, simulation, reporting, JWT auth, and API middleware.

---

## API Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/health` | — | Health check |
| `POST` | `/api/auth/register` | — | Create account |
| `POST` | `/api/auth/login` | — | Login, receive JWT tokens |
| `POST` | `/api/auth/refresh` | — | Refresh access token |
| `POST` | `/api/analyze` | Bearer | Upload CSVs, run analysis, persist report |
| `GET` | `/api/reports` | Bearer | List saved reports (paginated) |
| `GET` | `/api/reports/{id}` | Bearer | Get full report with results |
| `GET` | `/api/reports/{id}/csv` | Bearer | Download report as CSV |
| `DELETE` | `/api/reports/{id}` | Bearer | Delete a report |
| `GET` | `/api/user/profile` | Bearer | Get authenticated user profile |

Rate limited to 60 requests/minute per user. Full OpenAPI spec: [docs/openapi.yaml](docs/openapi.yaml).

---

## Changelog

### v0.3.0

- **REST API** — full JSON API with 10 endpoints: register, login, token refresh, analyze, CRUD reports, CSV download, user profile.
- **JWT authentication** — access tokens (15 min) + refresh tokens (7 days), bcrypt password hashing.
- **PostgreSQL persistence** — users table, reports stored with JSONB for results, automatic schema migration on startup.
- **Rate limiting** — in-memory per-user token bucket (60 req/min).
- **OpenAPI 3.0 spec** — complete API documentation in `docs/openapi.yaml`.
- **Docker Compose** — one-command PostgreSQL 16 setup.
- **Graceful shutdown** — API server handles SIGTERM/SIGINT cleanly.
- **21 new tests** — auth (12) + API middleware/helpers (9), bringing total to 57.

### v0.2.0

- **Web interface** — upload CSVs through the browser, view per-SKU reports with plain-English recommendations, download CSV exports.
- **Engine package** — high-level orchestrator shared by CLI and web, eliminating duplicated logic.
- **Parser refactor** — `io.Reader`-based functions for HTTP upload support; file-based functions delegate to them.
- **Unit test suite** — 39 tests covering parser, demand, inventory, simulation, and reporting.
- **Dual-mode main** — `-web` flag to launch the web server; all original CLI flags still work.

### v0.1.0

- Initial release: CLI tool with CSV parsing, demand statistics, safety stock / ROP / EOQ, Monte-Carlo simulation, and CSV export.

---

## License

Private — all rights reserved.
