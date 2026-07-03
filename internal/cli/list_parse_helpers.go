// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch helpers for hand-authored API commands added between generator runs.

package cli

import (
	"encoding/json"
	"strings"
)

func parseJSONOrCSVList(raw string) (any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}
	if strings.HasPrefix(raw, "[") {
		var parsed any
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out, nil
}
