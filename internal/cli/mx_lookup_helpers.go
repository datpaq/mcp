// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: MX lookup gateway accepts a plural domains parameter on /mx-lookup.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func mxDomainsValue(domains, legacyDomain string) string {
	if strings.TrimSpace(domains) != "" {
		return domains
	}
	return legacyDomain
}

func mxDomainsQueryValue(domains, legacyDomain string) string {
	values, err := parseMXDomains(mxDomainsValue(domains, legacyDomain))
	if err == nil && len(values) > 0 {
		return strings.Join(values, ",")
	}
	return mxDomainsValue(domains, legacyDomain)
}

func parseMXDomains(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	if strings.HasPrefix(raw, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err != nil {
			return nil, err
		}
		return cleanMXDomains(arr), nil
	}

	if strings.HasPrefix(raw, `"`) {
		var single string
		if err := json.Unmarshal([]byte(raw), &single); err != nil {
			return nil, err
		}
		return cleanMXDomains([]string{single}), nil
	}

	parts := strings.Split(raw, ",")
	return cleanMXDomains(parts), nil
}

func cleanMXDomains(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func mxDomainsBody(raw string) ([]string, error) {
	values, err := parseMXDomains(raw)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no domains supplied")
	}
	return values, nil
}

func normalizeMXBody(body map[string]any) map[string]any {
	if body == nil {
		return nil
	}
	if _, ok := body["domains"]; ok {
		delete(body, "domain")
		return body
	}
	raw, ok := body["domain"]
	if !ok {
		return body
	}
	delete(body, "domain")
	switch v := raw.(type) {
	case string:
		if values, err := parseMXDomains(v); err == nil && len(values) > 0 {
			body["domains"] = values
		}
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				values = append(values, s)
			}
		}
		if values = cleanMXDomains(values); len(values) > 0 {
			body["domains"] = values
		}
	default:
		body["domains"] = raw
	}
	return body
}

func shouldPrintResponseWarning(w io.Writer, flags *rootFlags) bool {
	if flags == nil {
		return isTerminal(w)
	}
	return wantsHumanTable(w, flags)
}
