package helpers

import (
	"encoding/json"
)

func JsonParser(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(data, &result) // This converts floats into only float64 ...
	if err != nil {
		return nil, err
	}

	return result, nil
}

func JsonStringifier(data map[string]interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}
