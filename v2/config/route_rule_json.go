package config

import (
	"encoding/json"
	"fmt"
)

func (x *Outbound) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		value, exists := Outbound_value[name]
		if !exists {
			return fmt.Errorf("unknown outbound %q", name)
		}
		*x = Outbound(value)
		return nil
	}

	var value int32
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("invalid outbound: %w", err)
	}
	if _, exists := Outbound_name[value]; !exists {
		return fmt.Errorf("unknown outbound value %d", value)
	}
	*x = Outbound(value)
	return nil
}

func (x Outbound) MarshalJSON() ([]byte, error) {
	name, exists := Outbound_name[int32(x)]
	if !exists {
		return nil, fmt.Errorf("unknown outbound value %d", x)
	}
	return json.Marshal(name)
}
