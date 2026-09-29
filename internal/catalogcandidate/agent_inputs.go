package catalogcandidate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mbcoward3/grocery-router/internal/ingest"
)

// AgentPacket is the compact, credential-free input for one isolated Pi worker.
type AgentPacket struct {
	SourceID             string               `json:"source_id"`
	CandidateIndex       int                  `json:"candidate_index"`
	SourceURL            string               `json:"source_url"`
	Recipe               json.RawMessage      `json:"recipe"`
	ExistingGroceryItems []ingest.GroceryItem `json:"existing_grocery_items"`
}

// WriteAgentPackets creates compact, credential-free inputs for isolated Pi workers.
func WriteAgentPackets(sourceRun, output string, documents []ingest.Document) (int, error) {
	_, _, nodes, err := ReadSourceRun(sourceRun)
	if err != nil {
		return 0, err
	}
	itemsByKey := map[string]ingest.GroceryItem{}
	for _, document := range documents {
		for _, section := range document.IngredientSections {
			for _, ingredient := range section.Ingredients {
				itemsByKey[ingredient.GroceryItem.Key] = ingredient.GroceryItem
			}
		}
	}
	keys := make([]string, 0, len(itemsByKey))
	for key := range itemsByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]ingest.GroceryItem, 0, len(keys))
	for _, key := range keys {
		items = append(items, itemsByKey[key])
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return 0, err
	}
	for _, node := range nodes {
		packet := AgentPacket{SourceID: node.Artifact.Source.ID, CandidateIndex: node.Index, SourceURL: node.Artifact.Source.URL, Recipe: node.Recipe.Recipe.Raw, ExistingGroceryItems: items}
		data, err := json.MarshalIndent(packet, "", "  ")
		if err != nil {
			return 0, err
		}
		name := fmt.Sprintf("%s--%d.json", node.Artifact.Source.ID, node.Index)
		if err := os.WriteFile(filepath.Join(output, name), append(data, '\n'), 0o644); err != nil {
			return 0, err
		}
	}
	return len(nodes), nil
}
