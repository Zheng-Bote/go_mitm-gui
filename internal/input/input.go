// Package input provides abstractions for loading data from various source formats.
package input

import (
	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

// DataSource defines the interface for loading and converting input data to JSON.
type DataSource interface {
	Format() model.DataFormat
	Load(path string) (*model.LoadedData, error)
}

// NewDataSource returns the appropriate DataSource for the given format.
func NewDataSource(format model.DataFormat, tc *model.TopicConfig) DataSource {
	switch format {
	case model.FormatCSV:
		return &CSVSource{}
	case model.FormatExcel:
		return &ExcelSource{}
	case model.FormatKafka:
		if tc != nil && tc.KafkaInput != nil {
			return NewKafkaSource(tc.KafkaInput)
		}
		return nil
	default:
		return nil
	}
}
