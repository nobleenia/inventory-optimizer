package records

// ColumnConfig represents a single column in an Excel template.
type ColumnConfig struct {
Header    string `json:"header"`
Width     int    `json:"width"`
DataType  string `json:"data_type"` // e.g., "string", "number", "currency", "date", "formula"
Formula   string `json:"formula,omitempty"` // Excel formula string, e.g., "=C%d*D%d" (with row index placehoder)
Prefill   string `json:"prefill,omitempty"` // Which SKU field to prefill, e.g., "sku", "price"
}

// Template represents a pre-built Excel template.
type Template struct {
ID          string         `json:"id"`
Name        string         `json:"name"`
Description string         `json:"description"`
Columns     []ColumnConfig `json:"columns"`
HasSummary  bool           `json:"has_summary"`
}

// GetAvailableTemplates returns the predefined library of Excel templates.
func GetAvailableTemplates() []Template {
return []Template{
{
ID:          "daily-sales-log",
Name:        "Daily Sales Log",
Description: "Track daily sales transactions and auto-calculate revenue.",
HasSummary:  true,
Columns: []ColumnConfig{
{Header: "Date", Width: 15, DataType: "date"},
{Header: "SKU", Width: 20, DataType: "string", Prefill: "sku_id"},
{Header: "Product Name", Width: 30, DataType: "string", Prefill: "name"},
{Header: "Quantity Sold", Width: 15, DataType: "number"},
{Header: "Unit Price", Width: 15, DataType: "currency", Prefill: "selling_price"},
{Header: "Total Revenue", Width: 18, DataType: "formula", Formula: "=D%d*E%d"},
},
},
{
ID:          "inventory-valuation",
Name:        "Inventory Valuation",
Description: "Calculate the total financial value of your current on-hand stock.",
HasSummary:  true,
Columns: []ColumnConfig{
{Header: "SKU", Width: 20, DataType: "string", Prefill: "sku_id"},
{Header: "Product Name", Width: 30, DataType: "string", Prefill: "name"},
{Header: "Current Stock", Width: 15, DataType: "number", Prefill: "current_stock"},
{Header: "Unit Cost", Width: 15, DataType: "currency", Prefill: "unit_cost"},
{Header: "Total Value", Width: 18, DataType: "formula", Formula: "=C%d*D%d"},
},
},
{
ID:          "profit-margin-calculator",
Name:        "Profit Margin Calculator",
Description: "Analyze gross profit margins per product based on cost and selling price.",
HasSummary:  true,
Columns: []ColumnConfig{
{Header: "SKU", Width: 20, DataType: "string", Prefill: "sku_id"},
{Header: "Product Name", Width: 30, DataType: "string", Prefill: "name"},
{Header: "Unit Cost", Width: 15, DataType: "currency", Prefill: "unit_cost"},
{Header: "Selling Price", Width: 15, DataType: "currency", Prefill: "selling_price"},
{Header: "Gross Profit $", Width: 18, DataType: "formula", Formula: "=D%d-C%d"},
{Header: "Margin %", Width: 15, DataType: "formula", Formula: "=IF(D%d>0, E%d/D%d, 0)"},
},
},
{
ID:          "purchase-order-tracker",
Name:        "Purchase Order Tracker",
Description: "Manage incoming inventory, supplier costs, and delivery statuses.",
HasSummary:  false,
Columns: []ColumnConfig{
{Header: "PO Number", Width: 15, DataType: "string"},
{Header: "Order Date", Width: 15, DataType: "date"},
{Header: "SKU", Width: 20, DataType: "string", Prefill: "sku_id"},
{Header: "Order Qty", Width: 15, DataType: "number"},
{Header: "Unit Cost", Width: 15, DataType: "currency", Prefill: "unit_cost"},
{Header: "Total PO Value", Width: 18, DataType: "formula", Formula: "=D%d*E%d"},
{Header: "Status", Width: 15, DataType: "string"},
{Header: "Expected Arrival", Width: 18, DataType: "date"},
},
},
{
ID:          "stock-take-sheet",
Name:        "Stock Take Sheet",
Description: "Printable sheet for physical warehouse cycle counting.",
HasSummary:  false,
Columns: []ColumnConfig{
{Header: "SKU", Width: 20, DataType: "string", Prefill: "sku_id"},
{Header: "Product Name", Width: 30, DataType: "string", Prefill: "name"},
{Header: "System Qty", Width: 15, DataType: "number", Prefill: "current_stock"},
{Header: "Counted Qty", Width: 15, DataType: "number"},
{Header: "Variance", Width: 15, DataType: "formula", Formula: "=D%d-C%d"},
{Header: "Notes", Width: 30, DataType: "string"},
},
},
{
ID:          "expense-tracker",
Name:        "Expense Tracker",
Description: "Keep track of operational overhead, software, and shipping materials.",
HasSummary:  true,
Columns: []ColumnConfig{
{Header: "Date", Width: 15, DataType: "date"},
{Header: "Category", Width: 20, DataType: "string"},
{Header: "Description", Width: 30, DataType: "string"},
{Header: "Cost", Width: 15, DataType: "currency"},
{Header: "Payment Method", Width: 20, DataType: "string"},
{Header: "Receipt?", Width: 10, DataType: "string"},
},
},
{
ID:          "cash-flow-summary",
Name:        "Cash Flow Summary",
Description: "Monthly view comparing incoming revenue vs outgoing inventory/expenses.",
HasSummary:  true,
Columns: []ColumnConfig{
{Header: "Month", Width: 15, DataType: "string"},
{Header: "Total Sales", Width: 18, DataType: "currency"},
{Header: "COGS", Width: 18, DataType: "currency"},
{Header: "OpEx", Width: 18, DataType: "currency"},
{Header: "Net Cash Flow", Width: 18, DataType: "formula", Formula: "=B%d-C%d-D%d"},
},
},
}
}
