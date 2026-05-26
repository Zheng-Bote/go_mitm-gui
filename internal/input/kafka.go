package input

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

// KafkaSource implements DataSource for Kafka topics.
// It reads messages, extracts EmploymentInformation, flattens JobInformation
// to the top level, and returns all records as a JSON array.
type KafkaSource struct {
	config *model.KafkaInputConfig
}

// NewKafkaSource creates a KafkaSource with the given config.
func NewKafkaSource(config *model.KafkaInputConfig) *KafkaSource {
	return &KafkaSource{config: config}
}

func (s *KafkaSource) Format() model.DataFormat {
	return model.FormatKafka
}

// Load connects to Kafka, consumes messages until idle timeout,
// transforms them (EmploymentInformation flattening), and returns
// the records as a JSON array.
func (s *KafkaSource) Load(path string) (*model.LoadedData, error) {
	cfg := s.config
	if cfg == nil {
		return nil, fmt.Errorf("kafka: no configuration provided")
	}

	// Build SASL mechanism.
	saslMechanism := plain.Mechanism{
		Username: cfg.Key,
		Password: cfg.Secret,
	}

	// Set consumer timeout (default 30s).
	idleTimeout := time.Duration(cfg.ConsumerTimeout) * time.Second
	if idleTimeout <= 0 {
		idleTimeout = 30 * time.Second
	}

	// TLS dialer for Confluent Cloud.
	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		TLS:           &tls.Config{},
		SASLMechanism: saslMechanism,
	}

	// Create reader.
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{cfg.BootstrapServers},
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		Dialer:         dialer,
		StartOffset:    kafka.FirstOffset,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
	defer reader.Close()

	ctx := context.Background()
	var records []interface{}

	for {
		// Per-read timeout for idle detection.
		readCtx, cancel := context.WithTimeout(ctx, idleTimeout)

		msg, err := reader.ReadMessage(readCtx)
		cancel()

		if err != nil {
			// Idle timeout reached - no more messages.
			if err == context.DeadlineExceeded {
				break
			}
			return nil, fmt.Errorf("kafka: read error: %w", err)
		}

		// Parse the message value as JSON.
		var raw map[string]interface{}
		if err := json.Unmarshal(msg.Value, &raw); err != nil {
			// Skip non-JSON messages.
			continue
		}

		// Extract EmploymentInformation.
		ei, ok := raw["EmploymentInformation"]
		if !ok {
			continue
		}
		eiMap, ok := ei.(map[string]interface{})
		if !ok || len(eiMap) == 0 {
			continue
		}

		// Flatten: promote JobInformation fields to same level.
		record := make(map[string]interface{})
		for k, v := range eiMap {
			if k == "JobInformation" {
				// Promote JobInformation's sub-fields.
				if ji, ok := v.(map[string]interface{}); ok {
					for jk, jv := range ji {
						record[jk] = jv
					}
				}
			} else {
				record[k] = v
			}
		}

		records = append(records, record)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("kafka: no records found in topic %q", cfg.Topic)
	}

	payload, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("kafka: marshal error: %w", err)
	}

	return &model.LoadedData{
		Format:      model.FormatKafka,
		SourcePath:  fmt.Sprintf("kafka://%s/%s", cfg.BootstrapServers, cfg.Topic),
		RowCount:    len(records),
		JSONPayload: payload,
		Fields:      extractFields(records),
	}, nil
}

// extractFields collects all unique field names from the records.
func extractFields(records []interface{}) []string {
	seen := make(map[string]bool)
	var fields []string
	for _, rec := range records {
		recMap, ok := rec.(map[string]interface{})
		if !ok {
			continue
		}
		for k := range recMap {
			if !seen[k] {
				seen[k] = true
				fields = append(fields, k)
			}
		}
	}
	return fields
}
