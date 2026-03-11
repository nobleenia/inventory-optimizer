# Inventory Optimizer

A lightweight inventory decision engine for e-commerce sellers. It analyses historical sales data, computes optimal reorder policies, and simulates future inventory performance using Monte-Carlo methods.

**Version:** 0.1.0  
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

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
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
│   └── main.go              # CLI entry point & orchestration
├── internal/
│   ├── models/
│   │   └── sku.go            # Core data types shared across packages
│   ├── parser/
│   │   └── csv_reader.go     # CSV ingestion & validation
│   ├── demand/
│   │   └── statistics.go     # Demand statistical analysis
│   ├── inventory/
│   │   ├── eoq.go            # Economic Order Quantity
│   │   ├── safety_stock.go   # Safety stock (Z-score approach)
│   │   ├── reorder_point.go  # Reorder point
│   │   └── policy.go         # Unified policy computation
│   ├── simulation/
│   │   └── monte_carlo.go    # Monte-Carlo inventory simulation
│   └── reporting/
│       └── results.go        # CLI display & CSV export
└── data/
    ├── sales_history.csv     # Sample sales data
    └── sku_parameters.csv    # Sample SKU config
```

Each package has a **single responsibility** and communicates only through types defined in `models/`.

---

## License

Private — all rights reserved.
