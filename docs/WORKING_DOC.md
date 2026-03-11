# Inventory Decision Engine for E-commerce Sellers
### Working Paper / Project Specification

Author: *Noble ELUWAH*  
Language: Go (Golang)  
Project Type: Decision Support Tool  
Version Target: V1 (Functional Prototype)

---

# 1. Project Overview

## 1.1 Purpose

This project builds a **lightweight inventory decision engine** designed to help small and mid-sized e-commerce sellers determine:

- When to reorder products
- How much to reorder
- How much safety stock to maintain
- The financial impact of different inventory policies

The system uses **historical demand data and stochastic simulation** to estimate inventory performance under uncertainty.

The objective is not simply to compute formulas but to provide **operational decision support** that helps businesses reduce:

- Stockouts
- Excess inventory
- Working capital locked in stock
- Inventory holding costs

---

# 2. Core Objective

The core objective of this project is to build a **simple, reliable inventory optimization engine** capable of:

1. Analyzing historical demand
2. Estimating demand variability
3. Computing optimal reorder policies
4. Simulating future inventory behavior
5. Estimating financial impact

The engine should provide **clear recommendations per SKU**.

Example outputs:

- Recommended reorder point
- Recommended order quantity
- Safety stock level
- Expected stockouts
- Expected annual cost
- Potential savings compared to naive policies

---

# 3. Target Users

Primary users:

Small-to-medium **e-commerce sellers** with:

- 20 to 500 SKUs
- Spreadsheet-based inventory tracking
- Limited analytics capability
- High sensitivity to cash flow

Typical platforms:

- Shopify stores
- Amazon FBA sellers
- WooCommerce stores
- Independent online brands

Revenue range target:

€20,000 – €500,000 annual sales.

---

# 4. Problem Being Solved

E-commerce operators frequently face two costly problems:

### 4.1 Overstock

Too much capital tied up in inventory.

Consequences:

- Cash flow constraints
- Storage costs
- Dead stock

---

### 4.2 Stockouts

Running out of products before restocking.

Consequences:

- Lost sales
- Lower marketplace rankings
- Customer dissatisfaction

---

### 4.3 Poor Replenishment Decisions

Most small operators reorder based on intuition or simple rules like:

- “Order when inventory looks low”
- “Order every month”
- “Order the same quantity every time”

These rules ignore:

- Demand variability
- Lead time uncertainty
- Cost trade-offs

This project provides **data-driven decision support**.

---

# 5. System Scope

## 5.1 Maximum System Capacity

Version 1 will support:

- Maximum SKUs per dataset: **500**
- Maximum demand history: **2 years**
- Simulation horizon: **52 weeks**
- Monte Carlo simulations: **500 runs**

These limits ensure:

- Manageable computation
- Easier debugging
- Stable performance

---

# 6. Input Data Structure

Two CSV files are required.

---

## 6.1 Sales History File

File: `sales_history.csv`

| sku | week | units_sold |
|----|----|----|
| SKU001 | 2024-01-01 | 12 |
| SKU001 | 2024-01-08 | 15 |
| SKU002 | 2024-01-01 | 6 |

Rules:

- Weekly data
- Date format: `YYYY-MM-DD`
- Each row represents weekly demand

Missing weeks are allowed but assumed to have **zero demand**.

---

## 6.2 SKU Parameter File

File: `sku_parameters.csv`

| sku | current_inventory | lead_time_days | unit_cost | order_cost | holding_cost_rate |
|----|----|----|----|----|----|
| SKU001 | 120 | 21 | 8.5 | 40 | 0.25 |

Definitions:

Current inventory  
Current stock on hand.

Lead time  
Time between order placement and arrival.

Unit cost  
Cost per product unit.

Order cost  
Cost incurred per order (administrative + logistics).

Holding cost rate  
Annual carrying cost percentage.

Example:
```
0.25 = 25% annual holding cost
```
---

# 7. Core Calculations

The system computes several inventory control metrics.

---

## 7.1 Demand Statistics

For each SKU:

- Mean weekly demand
- Standard deviation of weekly demand
- Annual demand estimate

These metrics capture demand variability.

---

## 7.2 Demand Conversion

Weekly demand statistics are converted to daily equivalents.
```
daily_mean = weekly_mean / 7
daily_std = weekly_std / sqrt(7)
```

---

## 7.3 Lead Time Demand

Expected demand during supplier lead time.
```
mean_LT = daily_mean × lead_time_days
std_LT = daily_std × sqrt(lead_time_days)
```

---

## 7.4 Safety Stock

Safety stock protects against demand variability.

Using service level approach:
```
Safety Stock = Z × std_LT

Where Z corresponds to service level:

| Service Level | Z Value |
|----|----|
| 90% | 1.28 |
| 95% | 1.65 |
| 99% | 2.33 |

Version 1 default: **95% service level**.
```

---

## 7.5 Reorder Point (ROP)

The reorder point indicates when a replenishment order should be placed.
```
ROP = mean_LT + safety_stock
```

---

## 7.6 Economic Order Quantity (EOQ)

EOQ determines optimal order size minimizing ordering and holding costs.
```
EOQ = sqrt((2 × D × S) / H)

Where:

D = annual demand  
S = order cost  
H = annual holding cost per unit
```

---

# 8. Monte Carlo Simulation

Simulation models inventory behavior under uncertainty.

For each SKU:

- 500 simulation runs
- Each run simulates 52 weeks

Each week:

1. Generate random demand from normal distribution
2. Reduce inventory
3. Check reorder condition
4. Place order if inventory ≤ ROP
5. Deliver order after lead time

Simulation tracks:

- Stockout events
- Average inventory levels
- Total inventory cost

---

# 9. Output Metrics

For each SKU the system produces:

| Metric | Description |
|------|------|
| Reorder Point | Recommended reorder trigger |
| EOQ | Recommended order quantity |
| Safety Stock | Buffer inventory |
| Expected Stockouts | Estimated annual stockouts |
| Average Inventory | Mean inventory level |
| Annual Inventory Cost | Holding + ordering costs |

Outputs will be exportable to:

- CSV
- CLI display

Future versions may include PDF reports.

---

# 10. System Architecture

The project follows a modular architecture.
```
inventory-optimizer
│
├── cmd
│ └── main.go
│
├── internal
│
│ ├── models
│ │ └── sku.go
│
│ ├── parser
│ │ └── csv_reader.go
│
│ ├── demand
│ │ └── statistics.go
│
│ ├── inventory
│ │ ├── eoq.go
│ │ ├── safety_stock.go
│ │ └── reorder_point.go
│
│ ├── simulation
│ │ └── monte_carlo.go
│
│ └── reporting
│ └── results.go
│
└── data
```


The system should maintain separation between:

- data ingestion
- demand analysis
- optimization logic
- simulation
- reporting

---

# 11. What This Project Is

This project **is**:

- A decision support engine
- A practical inventory optimization tool
- A demonstration of supply chain analytics capability
- A foundation for consulting work
- A platform for further expansion

---

# 12. What This Project Is NOT

This project is **not**:

- A full ERP system
- A warehouse management system
- An AI forecasting platform
- A complex enterprise supply chain optimizer
- A SaaS product (at this stage)

It is intentionally **focused and minimal**.

---

# 13. Development Phases

---

# Phase 1 — Data Engine

Goal: Load and structure input data.

Tasks:

- Build CSV parser
- Create SKU data structures
- Validate input data
- Handle missing weeks

Output:

Structured SKU dataset.

---

# Phase 2 — Demand Analysis

Goal: Compute demand statistics.

Tasks:

- Mean demand calculation
- Standard deviation calculation
- Annual demand estimation

Output:

Statistical demand model per SKU.

---

# Phase 3 — Inventory Optimization

Goal: Generate reorder policies.

Tasks:

- Safety stock calculation
- Reorder point computation
- EOQ calculation

Output:

Inventory policy parameters.

---

# Phase 4 — Simulation Engine

Goal: Evaluate inventory performance.

Tasks:

- Build Monte Carlo simulation
- Model weekly demand randomness
- Track inventory evolution
- Track stockouts and costs

Output:

Performance estimates.

---

# Phase 5 — Reporting

Goal: Communicate results clearly.

Tasks:

- Create CLI output
- Export CSV summary
- Generate SKU performance tables

Output:

Decision-ready report.

---

# Phase 6 — Optional UI

Goal: Improve usability.

Possible additions:

- Simple web interface
- CSV upload portal
- Results dashboard

Not required for version 1.

---

# 14. Success Criteria

Version 1 is successful if:

1. CSV data loads correctly
2. Demand statistics are computed
3. Reorder policies are generated
4. Simulation runs successfully
5. Output results are interpretable

Even a **CLI-only working tool** qualifies as success.

---

# 15. Long-Term Expansion Possibilities

Future extensions may include:

- Demand forecasting models
- Multi-warehouse systems
- Multi-echelon inventory
- Supplier reliability modeling
- Transportation constraints
- Cost optimization across networks

These belong to later versions.

---

# 16. Strategic Purpose

Beyond the tool itself, this project serves as:

- A **portfolio project**
- A **consulting capability demonstration**
- A **decision-engine prototype**
- A stepping stone toward larger supply chain analytics tools

It demonstrates the ability to combine:

- supply chain knowledge
- operations research
- statistical modeling
- software engineering

---

# 17. Guiding Principle

The system should prioritize:

- correctness
- clarity
- simplicity
- practical usefulness

Not complexity.

A **working decision tool** is more valuable than an unfinished sophisticated system.

---

# End of Working Paper