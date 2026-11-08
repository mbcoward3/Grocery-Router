package stage

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/mbcoward3/grocery-router/internal/catalog/extract"
	"github.com/mbcoward3/grocery-router/internal/catalog/ingeststore"
	"github.com/mbcoward3/grocery-router/internal/store"
)

// ReplayOptions scopes a replay run.
type ReplayOptions struct {
	// Site limits replay to one site ID; empty means all evidence.
	Site string
}

// ReplaySummary is the result of a replay run.
type ReplaySummary struct {
	RunID int64
	// Evidence is how many evidence rows were re-extracted.
	Evidence int
	// Matched evidence re-extracted to exactly its stored blocks and nodes.
	Matched int
	// Backfilled evidence had no stored extraction; it was stored now.
	Backfilled int
	// Mismatched lists evidence IDs whose re-extraction differs.
	Mismatched []int64
	// Nodes is the number of source nodes produced by re-extraction.
	Nodes int
}

// ErrReplayMismatch means re-extraction differed from stored rows.
var ErrReplayMismatch = errors.New("replayed extraction differs from stored source nodes")

// Replay re-extracts stored evidence without network access. Stored blocks
// and source nodes are immutable, so replay compares them with the fresh
// extraction (block position, text and parse status; node block, location,
// @id and content digest) and stores an extraction only for evidence that has
// none. Any difference fails the run with ErrReplayMismatch.
func Replay(ctx context.Context, config Config, options ReplayOptions) (ReplaySummary, error) {
	var summary ReplaySummary
	err := withLock(ctx, config.DB, lockFetch, func() error {
		current, err := startRun(ctx, config.DB, "catalog-fetch", options.Site,
			map[string]any{"site": options.Site, "replay": true})
		if err != nil {
			return err
		}
		summary.RunID = current.id
		runErr := replay(ctx, config, current, options, &summary)
		return current.finish(ctx, runErr)
	})
	return summary, err
}

func replay(ctx context.Context, config Config, current *run, options ReplayOptions, summary *ReplaySummary) error {
	queries := store.New(config.DB)
	hosts, err := queries.ListIngestSiteHosts(ctx)
	if err != nil {
		return fmt.Errorf("list site hosts: %w", err)
	}
	hostsBySite := map[string][]string{}
	for _, host := range hosts {
		hostsBySite[host.SiteID] = append(hostsBySite[host.SiteID], host.Host)
	}
	var after int64
	for {
		rows, err := queries.ListIngestEvidenceForReplay(ctx, store.ListIngestEvidenceForReplayParams{
			AfterID: after, SiteID: sql.NullString{String: options.Site, Valid: options.Site != ""}, RowLimit: 50,
		})
		if err != nil {
			return fmt.Errorf("list evidence: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			after = row.ID
			if err := replayOne(ctx, config, queries, row, hostsBySite[row.SiteID], summary); err != nil {
				return err
			}
		}
	}
	current.counters.Add("evidence", int64(summary.Evidence))
	current.counters.Add("matched", int64(summary.Matched))
	current.counters.Add("backfilled", int64(summary.Backfilled))
	current.counters.Add("mismatched", int64(len(summary.Mismatched)))
	current.counters.Add("source_nodes", int64(summary.Nodes))
	if len(summary.Mismatched) > 0 {
		return fmt.Errorf("%w: evidence %v", ErrReplayMismatch, summary.Mismatched)
	}
	return nil
}

func replayOne(ctx context.Context, config Config, queries *store.Queries, row store.IngestEvidence, hosts []string, summary *ReplaySummary) error {
	record, err := ingeststore.FromRow(row)
	if err != nil {
		return err
	}
	fresh, err := extract.FromEvidence(record, hosts)
	if err != nil {
		return fmt.Errorf("re-extract evidence %d: %w", row.ID, err)
	}
	summary.Evidence++
	summary.Nodes += len(fresh.Nodes)
	blocks, err := queries.ListIngestJSONLDBlocks(ctx, row.ID)
	if err != nil {
		return fmt.Errorf("list blocks of evidence %d: %w", row.ID, err)
	}
	nodes, err := queries.ListIngestSourceNodesForEvidence(ctx, row.ID)
	if err != nil {
		return fmt.Errorf("list nodes of evidence %d: %w", row.ID, err)
	}
	if len(blocks) == 0 && len(nodes) == 0 && len(fresh.Blocks) > 0 {
		if err := (&ingeststore.Extractions{Queries: queries, SiteID: row.SiteID}).SaveExtraction(ctx, row.ID, fresh); err != nil {
			return fmt.Errorf("backfill extraction of evidence %d: %w", row.ID, err)
		}
		summary.Backfilled++
		return nil
	}
	if sameExtraction(blocks, nodes, fresh) {
		summary.Matched++
		return nil
	}
	summary.Mismatched = append(summary.Mismatched, row.ID)
	config.progress("replay: evidence %d differs from its stored extraction", row.ID)
	return nil
}

func sameExtraction(blocks []store.IngestEvidenceJsonldBlock, nodes []store.IngestSourceNode, fresh extract.Result) bool {
	if len(blocks) != len(fresh.Blocks) || len(nodes) != len(fresh.Nodes) {
		return false
	}
	positionByID := map[int64]int{}
	for index, block := range blocks {
		want := fresh.Blocks[index]
		status := "parsed"
		if want.ParseError != "" {
			status = "invalid_json"
		}
		if int(block.Position) != want.Index || block.RawText != ingeststore.CleanText(want.Raw) || block.ParseStatus != status {
			return false
		}
		positionByID[block.ID] = int(block.Position)
	}
	type key struct {
		block    int
		location string
		id       string
		digest   string
	}
	stored := map[key]int{}
	for _, node := range nodes {
		stored[key{positionByID[node.JsonldBlockID], node.Location, node.NodeID.String, hex.EncodeToString(node.ContentDigest)}]++
	}
	for _, node := range fresh.Nodes {
		k := key{node.BlockIndex, node.Location, node.ID, node.ContentDigest}
		if stored[k] == 0 {
			return false
		}
		stored[k]--
	}
	return true
}
