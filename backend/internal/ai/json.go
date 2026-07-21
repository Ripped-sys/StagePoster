package ai

import (
	"encoding/json"
	"errors"
	"strings"
)

func DecodeJSONObject(
	raw string,
	output any,
) error {

	raw = strings.TrimSpace(raw)

	raw = strings.TrimPrefix(
		raw,
		"```json",
	)

	raw = strings.TrimPrefix(
		raw,
		"```",
	)

	raw = strings.TrimSuffix(
		raw,
		"```",
	)

	raw = strings.TrimSpace(raw)

	start :=
		strings.Index(
			raw,
			"{",
		)

	end :=
		strings.LastIndex(
			raw,
			"}",
		)

	if start < 0 || end < start {
		return errors.New(
			"json object not found",
		)
	}

	return json.Unmarshal(
		[]byte(
			raw[start:end+1],
		),
		output,
	)
}
