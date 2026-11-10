package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/settings"
	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/tenant"
)

// householdTimezoneCommands is embedded into cli so the household-timezone command lives here.
type householdTimezoneCommands struct {
	HouseholdTimezone householdTimezoneCommand `cmd:"" name:"household-timezone" help:"Set the IANA time zone that dates a household's plans."`
}

type householdTimezoneCommand struct {
	DatabaseConfig
	Zone      string `arg:"" help:"IANA time zone name, for example America/Chicago."`
	Household string `help:"Household ID." default:"${coward_household_id}"`
}

func (command *householdTimezoneCommand) Run() error {
	// The zone is validated before any database access, so a bad zone changes nothing.
	if _, err := settings.ValidateTimezone(command.Zone); err != nil {
		return err
	}
	if !householdIDPattern.MatchString(command.Household) {
		return fmt.Errorf("household ID %q is not a UUID", command.Household)
	}
	if err := migrate(command.DatabaseURL); err != nil {
		return err
	}
	db, err := database.Open(command.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx := context.Background()
	// The same change as the Settings screen.
	_, err = settings.Update(ctx, store.New(db), command.Household, settings.Change{Timezone: &command.Zone})
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("household %s not found", command.Household)
	}
	if err != nil {
		return fmt.Errorf("set household time zone: %w", err)
	}
	name, err := store.New(db).GetHouseholdName(ctx, command.Household)
	if err != nil {
		return fmt.Errorf("read household name: %w", err)
	}
	fmt.Printf("household %s time zone set to %s\n", name, command.Zone)
	return nil
}

// householdIDPattern accepts the canonical UUID form that household IDs use.
var householdIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// cliVars supplies kong defaults that come from Go constants.
var cliVars = map[string]string{"coward_household_id": tenant.CowardHouseholdID}
