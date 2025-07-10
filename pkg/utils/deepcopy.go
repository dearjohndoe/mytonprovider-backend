package utils

import (
	"encoding/json"
)

// DeepCopy creates a deep copy of the provided source object using JSON serialization.
// Not fast, so use accurately.
func DeepCopy[T any](src T) (T, error) {
	var dst T

	data, err := json.Marshal(src)
	if err != nil {
		return dst, err
	}

	err = json.Unmarshal(data, &dst)
	if err != nil {
		return dst, err
	}

	return dst, nil
}

func MustDeepCopy[T any](src T) T {
	dst, err := DeepCopy(src)
	if err != nil {
		panic(err)
	}
	return dst
}
