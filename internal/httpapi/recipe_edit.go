package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/week"
)

// recipeEditRequest is a household recipe's complete new content: RecipeDetail without IDs,
// sources, or derived fields. Each ingredient names an existing grocery item or a new one.
type recipeEditRequest struct {
	Name                string                         `json:"name"`
	ImageURL            *string                        `json:"imageUrl"`
	Yield               *string                        `json:"yield"`
	HandsOn             *durationRange                 `json:"handsOn"`
	Unattended          *durationRange                 `json:"unattended"`
	IngredientSections  []ingredientSectionEdit        `json:"ingredientSections"`
	InstructionSections []instructionSectionEditFields `json:"instructionSections"`
}

type ingredientSectionEdit struct {
	Name        string           `json:"name"`
	Ingredients []ingredientEdit `json:"ingredients"`
}

type ingredientEdit struct {
	SourceText     string          `json:"sourceText"`
	Quantity       recipeQuantity  `json:"quantity"`
	Preparation    *string         `json:"preparation"`
	Optional       bool            `json:"optional"`
	DisplayNote    *string         `json:"displayNote"`
	OnGroceryList  bool            `json:"onGroceryList"`
	GroceryItemID  *int64          `json:"groceryItemId"`
	NewGroceryItem *newGroceryItem `json:"newGroceryItem"`
}

type newGroceryItem struct {
	Name           string `json:"name"`
	StoreSectionID int64  `json:"storeSectionId"`
	ShoppingMode   string `json:"shoppingMode"`
}

type instructionSectionEditFields struct {
	Name  string           `json:"name"`
	Steps []stepEditFields `json:"steps"`
}

type stepEditFields struct {
	Instruction string `json:"instruction"`
}

type fieldProblem struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type invalidRecipeEnvelope struct {
	Error  apiError       `json:"error"`
	Fields []fieldProblem `json:"fields"`
}

// editRecipe replaces a household recipe's content. The save is the household's approval, and
// the current plan's list is rebuilt when it contains the recipe. Past plans never change.
func (server *Server) editRecipe(response http.ResponseWriter, request *http.Request) {
	recipeID, ok := pathID(response, request, "recipeID")
	if !ok {
		return
	}
	var userID string
	if server.currentUser != nil {
		userID, ok = server.currentUser(request.Context())
	}
	if !ok || userID == "" {
		writeError(response, http.StatusUnauthorized, "authentication_required", "Sign in is required.")
		return
	}
	var input recipeEditRequest
	if !decodeJSON(response, request, &input) {
		return
	}
	edit, problems := recipeEditFromRequest(input)
	if len(problems) > 0 {
		writeInvalidRecipe(response, problems)
		return
	}
	edit.VerifiedBy, edit.Now = userID, server.now()

	householdID := request.PathValue("householdID")
	err := server.replaceRecipe(request.Context(), householdID, recipeID, edit)
	var contentErr *ingest.ContentError
	switch {
	case errors.Is(err, ingest.ErrRecipeNotFound):
		writeError(response, http.StatusNotFound, "recipe_not_found", "The recipe was not found.")
		return
	case errors.As(err, &contentErr):
		for _, field := range contentErr.Fields {
			problems = append(problems, fieldProblem{Path: requestPath(field.Path), Message: field.Message})
		}
		writeInvalidRecipe(response, problems)
		return
	case err != nil:
		writeInternalError(response, request, err)
		return
	}
	detail, err := server.recipeDetail(request.Context(), householdID, recipeID)
	if err != nil {
		writeInternalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, detail)
}

// replaceRecipe applies an edit and the current-plan rebuild in one transaction.
func (server *Server) replaceRecipe(ctx context.Context, householdID string, recipeID int64, edit ingest.RecipeEdit) error {
	tx, err := server.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	queries := store.New(tx)
	// Every plan change locks the open plan before anything else; taking it before the recipe
	// keeps that order, so an edit and a plan change wait for each other instead of deadlocking.
	if _, err := queries.LockOpenWeek(ctx, householdID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err := ingest.ReplaceRecipe(ctx, queries, householdID, recipeID, edit); err != nil {
		return err
	}
	if _, err := week.RecipeChanged(ctx, queries, householdID, recipeID); err != nil {
		return err
	}
	return tx.Commit()
}

func writeInvalidRecipe(response http.ResponseWriter, problems []fieldProblem) {
	writeJSON(response, http.StatusUnprocessableEntity, invalidRecipeEnvelope{
		Error:  apiError{Code: "invalid_recipe", Message: "The recipe was not saved. Fix the marked fields."},
		Fields: problems,
	})
}

// recipeEditFromRequest converts the request to recipe content. Problems the content format
// cannot express (a duration with one end, a line with two item choices) are reported here.
func recipeEditFromRequest(input recipeEditRequest) (ingest.RecipeEdit, []fieldProblem) {
	var problems []fieldProblem
	duration := func(path string, value *durationRange) *ingest.Duration {
		if value == nil || (value.MinimumMinutes == nil && value.MaximumMinutes == nil) {
			return nil
		}
		if value.MinimumMinutes == nil || value.MaximumMinutes == nil {
			problems = append(problems, fieldProblem{Path: path, Message: "enter both minimum and maximum minutes, or neither"})
			return nil
		}
		return &ingest.Duration{Min: *value.MinimumMinutes, Max: *value.MaximumMinutes}
	}
	edit := ingest.RecipeEdit{Content: ingest.RecipeContent{
		Name:       input.Name,
		ImageURL:   text(input.ImageURL),
		Yield:      text(input.Yield),
		HandsOn:    duration("handsOn", input.HandsOn),
		Unattended: duration("unattended", input.Unattended),
	}}
	for sectionIndex, section := range input.IngredientSections {
		ingredients := make([]ingest.Ingredient, 0, len(section.Ingredients))
		choices := make([]ingest.ItemChoice, 0, len(section.Ingredients))
		for ingredientIndex, line := range section.Ingredients {
			ingredients = append(ingredients, ingest.Ingredient{
				SourceText:  line.SourceText,
				Quantity:    contentQuantity(line.Quantity),
				Preparation: text(line.Preparation),
				Optional:    line.Optional,
				NonShopping: !line.OnGroceryList,
				Note:        text(line.DisplayNote),
			})
			var choice ingest.ItemChoice
			switch {
			case line.GroceryItemID != nil && line.NewGroceryItem != nil:
				problems = append(problems, fieldProblem{
					Path:    fmt.Sprintf("ingredientSections[%d].ingredients[%d].groceryItemId", sectionIndex, ingredientIndex),
					Message: "choose an existing grocery item or a new one, not both",
				})
			case line.GroceryItemID != nil:
				choice.GroceryItemID = *line.GroceryItemID
			case line.NewGroceryItem != nil:
				choice.New = &ingest.NewGroceryItem{
					Name: line.NewGroceryItem.Name, StoreSectionID: line.NewGroceryItem.StoreSectionID,
					ShoppingMode: line.NewGroceryItem.ShoppingMode,
				}
			}
			choices = append(choices, choice)
		}
		edit.Content.IngredientSections = append(edit.Content.IngredientSections, ingest.IngredientSection{Name: section.Name, Ingredients: ingredients})
		edit.Items = append(edit.Items, choices)
	}
	for _, section := range input.InstructionSections {
		steps := make([]string, 0, len(section.Steps))
		for _, step := range section.Steps {
			steps = append(steps, step.Instruction)
		}
		edit.Content.InstructionSections = append(edit.Content.InstructionSections, ingest.InstructionSection{Name: section.Name, Steps: steps})
	}
	return edit, problems
}

func contentQuantity(quantity recipeQuantity) ingest.Quantity {
	result := ingest.Quantity{
		Kind: quantity.Kind, Amount: text(quantity.Amount), Maximum: text(quantity.Maximum), Unit: text(quantity.Unit),
	}
	if quantity.Package != nil {
		result.Package = &ingest.Package{Type: quantity.Package.Type, Size: text(quantity.Package.Size), Unit: text(quantity.Package.Unit)}
	}
	return result
}

func text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// requestFieldNames maps the content format's field names to the request's.
var requestFieldNames = map[string]string{
	"note":         "displayNote",
	"grocery_item": "groceryItemId",
}

// requestPath rewrites a content field path (ingredient_sections[0].ingredients[1].source_text)
// as a request path (ingredientSections[0].ingredients[1].sourceText).
func requestPath(path string) string {
	segments := strings.Split(path, ".")
	for index, segment := range segments {
		name, suffix := segment, ""
		if bracket := strings.IndexByte(segment, '['); bracket >= 0 {
			name, suffix = segment[:bracket], segment[bracket:]
		}
		if mapped, ok := requestFieldNames[name]; ok {
			name = mapped
		} else {
			parts := strings.Split(name, "_")
			for part := 1; part < len(parts); part++ {
				if parts[part] != "" {
					parts[part] = strings.ToUpper(parts[part][:1]) + parts[part][1:]
				}
			}
			name = strings.Join(parts, "")
		}
		segments[index] = name + suffix
	}
	return strings.Join(segments, ".")
}
