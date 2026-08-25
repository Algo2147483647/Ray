package factory

import "encoding/json"

// specMap is an internal compiler representation used by the existing
// numerical builders. It is not a protocol or compatibility entry point.
func specMap(spec interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
