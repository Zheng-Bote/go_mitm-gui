package input

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

type CSVSource struct{}

func (s *CSVSource) Format() model.DataFormat { return model.FormatCSV }

func (s *CSVSource) Load(path string) (*model.LoadedData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("csv: cannot open %q: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv: failed to read %q: %w", path, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv: %q has no data rows", path)
	}

	headers := make([]string, len(records[0]))
	for i, h := range records[0] {
		headers[i] = strings.TrimSpace(h)
	}

	var rows []map[string]interface{}
	for rowIdx := 1; rowIdx < len(records); rowIdx++ {
		record := records[rowIdx]
		obj := make(map[string]interface{})
		for colIdx, header := range headers {
			if colIdx < len(record) {
				val := strings.TrimSpace(record[colIdx])
				if val != "" {
					obj[header] = val
				}
			}
		}
		if len(obj) > 0 {
			rows = append(rows, obj)
		}
	}

	payload, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("csv: failed to marshal JSON: %w", err)
	}

	return &model.LoadedData{
		Format:      model.FormatCSV,
		SourcePath:  path,
		RowCount:    len(rows),
		JSONPayload: payload,
		Fields:      headers,
	}, nil
}
