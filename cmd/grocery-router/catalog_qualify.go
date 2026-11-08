package main

import (
	"errors"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/mbcoward3/grocery-router/internal/catalog/qualify"
	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
)

// catalogQualifyCommands is embedded into cli so the catalog-qualify command lives in this file.
type catalogQualifyCommands struct {
	CatalogQualify catalogQualifyCommand `cmd:"" name:"catalog-qualify" help:"Score a candidate set against the evaluation set and holdout (spec §6.3)."`
}

type catalogQualifyCommand struct {
	Root                string `help:"Repository root." default:"." env:"GROCERY_ROUTER_ROOT" type:"path"`
	Registry            string `help:"Site registry path, relative to root." default:"catalog/sites.yaml" env:"GROCERY_ROUTER_CATALOG_SITES"`
	Eval                string `help:"Evaluation set directory, relative to root." default:"catalog/eval"`
	Candidates          string `help:"Directory of candidate files named <case-id>.yaml." required:"" type:"path"`
	Holdout             string `help:"Operator holdout directory, kept outside the repository. Qualification fails without it." type:"path"`
	AcceptedRegressions string `name:"accepted-regressions" help:"Accepted-regressions manifest file." type:"path"`
}

var errNotQualified = errors.New("candidate set did not qualify")

func (command *catalogQualifyCommand) Run(kongContext *kong.Context) error {
	sites, err := registry.ReadFile(filepath.Join(command.Root, filepath.FromSlash(command.Registry)))
	if err != nil {
		return err
	}
	input := qualify.Input{Sites: sites}
	if input.EvalSet, err = qualify.ReadEvalSet(filepath.Join(command.Root, filepath.FromSlash(command.Eval)), sites); err != nil {
		return err
	}
	if command.Holdout != "" {
		if input.Holdout, err = qualify.ReadHoldout(command.Holdout, sites); err != nil {
			return err
		}
	}
	if command.AcceptedRegressions != "" {
		if input.Accepted, err = qualify.ReadAcceptedRegressions(command.AcceptedRegressions); err != nil {
			return err
		}
	}
	if input.Candidates, err = qualify.ReadCandidateSet(command.Candidates); err != nil {
		return err
	}
	report, err := qualify.Qualify(input)
	if err != nil {
		return err
	}
	if err := report.WriteText(kongContext.Stdout); err != nil {
		return err
	}
	if !report.Qualified() {
		return errNotQualified
	}
	return nil
}
