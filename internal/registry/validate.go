package registry

import (
	"fmt"
	"regexp"
)

// idPattern matches both site_id and device_id. device_id is used verbatim
// in the MQTT topic devices/{device_id}/telemetry and in the Mosquitto ACL
// pattern (mosquitto/config/acl.conf) — it must never contain '/', '+', '#'
// or other characters with topic-wildcard meaning.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func validateID(field, value string) error {
	if !idPattern.MatchString(value) {
		return fmt.Errorf("%s must be 1-64 chars of letters, digits, '-', '_': got %q", field, value)
	}
	return nil
}

func validateRequired(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func errNegative(field string) error {
	return fmt.Errorf("%s cannot be negative", field)
}
