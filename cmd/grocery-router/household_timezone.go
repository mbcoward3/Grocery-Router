package main

import (
	"context"
	"fmt"

	"github.com/mbcoward3/grocery-router/internal/database"
	"github.com/mbcoward3/grocery-router/internal/settings"
	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/tenant"
)

// householdTimezoneCommands is embedded into cli so the household-timezone command lives here.
type householdTimezoneCommands struct {
	HouseholdTimezone householdTimezoneCommand `cmd:"" name:"household-timezone" help:"Set the IANA time zone that decides when a household's week begins."`
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
	if err := migrate(command.DatabaseURL); err != nil {
		return err
	}
	db, err := database.Open(command.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := store.New(tx).SetHouseholdTimezone(ctx, store.SetHouseholdTimezoneParams{
		HouseholdID: command.Household, Timezone: command.Zone,
	})
	if err != nil {
		return fmt.Errorf("set household time zone: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("household %s not found", command.Household)
	}
	var name string
	if err := tx.QueryRowContext(ctx, "SELECT name FROM households WHERE id = $1", command.Household).Scan(&name); err != nil {
		return fmt.Errorf("read household name: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("household %s time zone set to %s\n", name, command.Zone)
	return nil
}

// cliVars supplies kong defaults that come from Go constants.
var cliVars = map[string]string{"coward_household_id": tenant.CowardHouseholdID}
