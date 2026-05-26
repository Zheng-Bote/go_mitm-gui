package input

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

type ExcelSource struct{}

func (s *ExcelSource) Format() model.DataFormat { return model.FormatExcel }

func (s *ExcelSource) Load(path string) (*model.LoadedData, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("excel: cannot open %q: %w", path, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("excel: %q has no sheets", path)
	}
	sheetName := sheets[0]

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("excel: failed to read sheet %q: %w", sheetName, err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("excel: %q sheet %q has no data rows", path, sheetName)
	}

	headers := make([]string, len(rows[0]))
	for i, h := range rows[0] {
		headers[i] = strings.TrimSpace(h)
	}

	var result []map[string]interface{}
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		obj := make(map[string]interface{})
		for colIdx, header := range headers {
			if colIdx < len(row) {
				val := strings.TrimSpace(row[colIdx])
				if val != "" {
					obj[header] = val
				}
			}
		}
		if len(obj) > 0 {
			result = append(result, obj)
		}
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("excel: failed to marshal JSON: %w", err)
	}

	return &model.LoadedData{
		Format:      model.FormatExcel,
		SourcePath:  path,
		RowCount:    len(result),
		JSONPayload: payload,
		Fields:      headers,
	}, nil
}
