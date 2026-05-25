package records

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/noble-ch/inventory-optimizer/internal/store"
	"github.com/xuri/excelize/v2"
)

var formulaReferencePattern = regexp.MustCompile(`([A-Z]+)%d`)

// GenerateExcel creates a new Excel file based on the template and provided active columns.
// If skus is provided, it logically pre-fills the data rows.
func GenerateExcel(tmpl Template, activeColumns []string, skus []store.SKU) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Data"
	f.SetSheetName("Sheet1", sheet)

	// Filter columns down to the active ones requested by the user, while
	// keeping formula columns and their source columns so the workbook stays valid.
	selected := make(map[string]struct{}, len(activeColumns))
	for _, header := range activeColumns {
		if header != "" {
			selected[header] = struct{}{}
		}
	}
	for _, col := range tmpl.Columns {
		if col.DataType != "formula" {
			continue
		}
		selected[col.Header] = struct{}{}
		for _, dependency := range formulaDependencies(col.Formula) {
			index, err := excelize.ColumnNameToNumber(dependency)
			if err != nil || index < 1 || index > len(tmpl.Columns) {
				continue
			}
			selected[tmpl.Columns[index-1].Header] = struct{}{}
		}
	}

	var columns []ColumnConfig
	var sourceIndexes []int
	for i, col := range tmpl.Columns {
		if _, ok := selected[col.Header]; ok {
			columns = append(columns, col)
			sourceIndexes = append(sourceIndexes, i+1)
		}
	}
	if len(columns) == 0 {
		columns = append(columns, tmpl.Columns...)
		for i := range tmpl.Columns {
			sourceIndexes = append(sourceIndexes, i+1)
		}
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns selected")
	}

	outputIndexBySourceIndex := make(map[int]int, len(sourceIndexes))
	for outputIndex, sourceIndex := range sourceIndexes {
		outputIndexBySourceIndex[sourceIndex] = outputIndex + 1
	}

	// 1. Write Headers & Set Column Widths
	for i, col := range columns {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s1", colName)
		f.SetCellValue(sheet, cell, col.Header)
		f.SetColWidth(sheet, colName, colName, float64(col.Width))
	}

	// Header Styling
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#0056b3"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	endColName, _ := excelize.ColumnNumberToName(len(columns))
	f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", endColName), style)

	// 2. Pre-fill rows if SKUs exist
	startRow := 2
	rowsToWrite := len(skus)
	if rowsToWrite == 0 {
		// Provide 50 empty rows if no SKUs to allow formulas to exist
		rowsToWrite = 50
	}

	currencyStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 164}) // Built-in accounting format

	for r := 0; r < rowsToWrite; r++ {
		rowIdx := startRow + r

		var sku store.SKU
		hasSKU := r < len(skus)
		if hasSKU {
			sku = skus[r]
		}

		for c, col := range columns {
			colName, _ := excelize.ColumnNumberToName(c + 1)
			cell := fmt.Sprintf("%s%d", colName, rowIdx)

			// Pre-fill data
			if hasSKU && col.Prefill != "" {
				val := getSKUFieldValue(sku, col.Prefill)
				if val != nil {
					f.SetCellValue(sheet, cell, val)
				}
			}

			// Apply formulas
			if col.DataType == "formula" && col.Formula != "" {
				formulaStr := rewriteFormula(col.Formula, rowIdx, outputIndexBySourceIndex)
				f.SetCellFormula(sheet, cell, formulaStr)
			}

			// Apply currency formatting styling
			if col.DataType == "currency" || col.DataType == "formula" && strings.Contains(col.Header, "$") || strings.Contains(col.Header, "Cost") || strings.Contains(col.Header, "Price") || strings.Contains(col.Header, "Value") {
				f.SetCellStyle(sheet, cell, cell, currencyStyle)
			}
		}
	}

	// 3. Conditional Formatting - e.g. for Inventory Valuation or Variance
	// Just a simple safety net conditional format to show off the capability
	varianceCol := getColumnNameByHeader(columns, "Variance")
	if varianceCol != "" {
		formatRule := []excelize.ConditionalFormatOptions{{
			Type:     "cell",
			Criteria: "<",
			Value:    "0",
			Format: func() *int {
				i, _ := f.NewConditionalStyle(&excelize.Style{Font: &excelize.Font{Color: "#9C0006"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFC7CE"}, Pattern: 1}})
				return &i
			}(),
		}}
		f.SetConditionalFormat(sheet, fmt.Sprintf("%s2:%s%d", varianceCol, varianceCol, startRow+rowsToWrite), formatRule)
	}

	// 4. Create Summary Sheet if Requested
	if tmpl.HasSummary {
		buildSummarySheet(f, "Summary KPIs", sheet, columns, startRow+rowsToWrite-1)
	}

	return f, nil
}

func buildSummarySheet(f *excelize.File, summarySheet string, dataSheet string, columns []ColumnConfig, maxDataRow int) {
	f.NewSheet(summarySheet)

	f.SetCellValue(summarySheet, "A1", "Metric")
	f.SetCellValue(summarySheet, "B1", "Value")
	f.SetColWidth(summarySheet, "A", "A", 25)
	f.SetColWidth(summarySheet, "B", "B", 20)

	style, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	f.SetCellStyle(summarySheet, "A1", "B1", style)

	summaryRow := 2
	for i, col := range columns {
		if col.DataType == "currency" || col.DataType == "number" || col.DataType == "formula" {
			colName, _ := excelize.ColumnNumberToName(i + 1)

			f.SetCellValue(summarySheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("Total %s", col.Header))

			// e.g. =SUM(Data!D2:D50)
			formula := fmt.Sprintf("=SUM('%s'!%s2:%s%d)", dataSheet, colName, colName, maxDataRow)
			f.SetCellFormula(summarySheet, fmt.Sprintf("B%d", summaryRow), formula)

			summaryRow++
		}
	}
}

func getSKUFieldValue(sku store.SKU, prefillKey string) interface{} {
	switch prefillKey {
	case "sku_id":
		return sku.SKUID
	case "name":
		return sku.Name
	case "unit_cost":
		return sku.UnitCost
	case "selling_price":
		return sku.SellingPrice
	case "current_stock":
		return sku.CurrentStock
	}
	return nil
}

func formulaDependencies(formula string) []string {
	matches := formulaReferencePattern.FindAllStringSubmatch(formula, -1)
	dependencies := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		letter := match[1]
		if _, ok := seen[letter]; ok {
			continue
		}
		seen[letter] = struct{}{}
		dependencies = append(dependencies, letter)
	}
	return dependencies
}

func rewriteFormula(formula string, rowIdx int, outputIndexBySourceIndex map[int]int) string {
	return formulaReferencePattern.ReplaceAllStringFunc(formula, func(match string) string {
		parts := formulaReferencePattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		sourceIndex, err := excelize.ColumnNameToNumber(parts[1])
		if err != nil {
			return match
		}

		outputIndex, ok := outputIndexBySourceIndex[sourceIndex]
		if !ok {
			outputIndex = sourceIndex
		}

		columnName, err := excelize.ColumnNumberToName(outputIndex)
		if err != nil {
			return match
		}

		return fmt.Sprintf("%s%d", columnName, rowIdx)
	})
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func getColumnNameByHeader(columns []ColumnConfig, header string) string {
	for i, col := range columns {
		if col.Header == header {
			name, _ := excelize.ColumnNumberToName(i + 1)
			return name
		}
	}
	return ""
}
