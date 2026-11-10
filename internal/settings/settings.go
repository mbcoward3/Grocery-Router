// Package settings validates household-configurable settings.
package settings

import (
	"errors"
	"fmt"
	"time"
)

// ValidateTimezone returns the IANA location for name. It rejects the empty string and "Local",
// which time.LoadLocation would otherwise resolve to UTC and the server's own zone.
func ValidateTimezone(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, errors.New("time zone must be an IANA zone name such as America/Chicago")
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone %q", name)
	}
	return location, nil
}
