package settings_test

import (
	"testing"
	_ "time/tzdata"

	"github.com/mbcoward3/grocery-router/internal/settings"
)

func TestValidateTimezone(t *testing.T) {
	for _, name := range []string{"America/Chicago", "UTC"} {
		location, err := settings.ValidateTimezone(name)
		if err != nil {
			t.Fatalf("ValidateTimezone(%q): %v", name, err)
		}
		if location.String() != name {
			t.Fatalf("ValidateTimezone(%q) = %q", name, location)
		}
	}
	for _, name := range []string{"", "Local", "Not/AZone"} {
		if _, err := settings.ValidateTimezone(name); err == nil {
			t.Fatalf("ValidateTimezone(%q) unexpectedly succeeded", name)
		}
	}
}
