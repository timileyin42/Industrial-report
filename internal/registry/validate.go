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

// countryPattern is deliberately just "2 uppercase letters," not a closed
// enum of real ISO 3166-1 alpha-2 codes — the emission-factor country
// field (migrations/0005_emission_factor.sql) is the same free-text
// shape, and keeping the two in sync means a typo shows up as "no
// emission factor configured for XX" (an honest 409) rather than a
// confusing validation error at site-creation time.
var countryPattern = regexp.MustCompile(`^[A-Z]{2}$`)

func validateID(field, value string) error {
	if !idPattern.MatchString(value) {
		return fmt.Errorf("%s must be 1-64 chars of letters, digits, '-', '_': got %q", field, value)
	}
	return nil
}

func validateCountry(field, value string) error {
	if !countryPattern.MatchString(value) {
		return fmt.Errorf("%s must be a 2-letter uppercase country code (e.g. NG, GB): got %q", field, value)
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
