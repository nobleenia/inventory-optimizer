# Inventory Optimizer

A lightweight inventory decision engine for e-commerce sellers. It analyses historical sales data, computes optimal reorder policies, and simulates future inventory performance using Monte-Carlo methods — available as both a CLI tool and a browser-based web application.

**Version:** 0.2.0  
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

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-web` | `false` | Launch the web interface instead of CLI mode |
| `-port` | `:8080` | Port for the web server (web mode only) |
| `-sales` | `data/sales_history.csv` | Path to weekly sales history CSV |
| `-params` | `data/sku_parameters.csv` | Path to SKU parameters CSV |
| `-output` | *(none)* | Path for CSV export (omit to skip) |
| `-service-level` | `0.95` | Target service level (0.90, 0.95, 0.99) |
| `-sim-runs` | `500` | Monte-Carlo simulation runs per SKU |
| `-sim-weeks` | `52` | Simulation horizon in weeks |
| `-version` | — | Print version and exit |

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
│   └── main.go                  # Dual-mode entry point (CLI / web)
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
│   └── web/
│       ├── server.go            # HTTP server, routes & handlers
│       ├── templates/           # Embedded HTML templates
│       │   ├── index.html
│       │   ├── results.html
│       │   └── error.html
│       └── static/
│           ├── css/style.css    # Responsive stylesheet
│           └── js/app.js        # File-input UX helpers
└── data/
    ├── sales_history.csv        # Sample sales data (3 SKUs × 52 weeks)
    └── sku_parameters.csv       # Sample SKU config
```

Each package has a **single responsibility** and communicates only through types defined in `models/`. The `engine` package orchestrates the full pipeline so that both CLI and web modes share one code path.

---

## Tests

```bash
go test ./... -v
```

39 unit tests across 5 packages covering parsing, statistics, inventory calculations, simulation determinism, and report output.

---

## Changelog

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
