package flow // import "github.com/orkestr8/xgraph/flow"

import (
	"encoding/json"
	"errors"
	"time"
)

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(((time.Duration)(d)).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		dd := time.Duration(value)
		*d = Duration(dd)
		return nil
	case string:
		var err error
		dd, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(dd)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type attributes struct {
	Inline     bool     `json:"inline,omitempty"`
	Timeout    Duration `json:"timeout,omitempty"`
	MaxWorkers int      `json:"max_workers,omitempty"`
	EdgeSorter string   `json:"edge_sorter,omitempty"`
}

// unmarshals a given map into the receiver's fields.
func (a *attributes) unmarshal(m map[string]interface{}) error {
	if m == nil {
		return nil
	}
	// just use json serialization and deserialization
	// to deal with the whole struct
	encoded, err := json.Marshal(m)
	if err != nil {
		return err
	}
	decoded := attributes{}
	err = json.Unmarshal(encoded, &decoded)
	if err != nil {
		return err
	}
	*a = decoded
	return nil
}
