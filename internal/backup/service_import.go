package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ImportRequest is what the HTTP handler hands to ImportBundle.
//
// Source must be a seekable buffer for archive/zip; HTTP multipart
// already gives us a *multipart.File which is seekable, but if the
// caller hands an arbitrary io.Reader we spool to a temp file
// first.
type ImportRequest struct {
	Source       io.Reader
	Filename     string
	RequestedBy  string
	Memories     bool
	Integrations bool
	CustomTasks  bool
}

// ImportBundle reads an export bundle (zip) and replays its
// contents into the live database.
//
// Conflict policy v1: id conflicts are SKIPPED (status counted in
// Skipped). Memories use ON CONFLICT (id) DO NOTHING; integrations
// rely on (id, route_prefix) uniqueness; custom_tasks on (id).
//
// Memories are inserted with embedder='imported_v1' and a NULL
// embedding column so Search() ignores them until the operator
// triggers re-embed via the Memory page's Maintenance tab. This
// keeps backup independent of memory's embedder configuration —
// import works regardless of what's running.
//
// Integrations are imported with enabled=false and an opaque
// api_key_hash placeholder ("imported:no-plaintext-key") so the
// row is visible but cannot authenticate. Operator must rotate
// the key explicitly (creating a new bcrypt hash) before use.
func (s *Service) ImportBundle(ctx context.Context, req ImportRequest) (Import, error) {
	if req.Source == nil {
		return Import{}, errors.New("import: source is nil")
	}
	if !req.Memories && !req.Integrations && !req.CustomTasks {
		return Import{}, errors.New("import: at least one entity scope must be selected")
	}
	if req.RequestedBy == "" {
		req.RequestedBy = "admin"
	}

	// archive/zip needs a ReaderAt; spool to a temp file first.
	if err := os.MkdirAll(s.cfg.LocalDir, 0o700); err != nil {
		return Import{}, fmt.Errorf("import: mkdir local_dir: %w", err)
	}
	tmp, err := os.CreateTemp(s.cfg.LocalDir, ".import-*.zip")
	if err != nil {
		return Import{}, fmt.Errorf("import: create tmp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	bytesIn, copyErr := io.Copy(tmp, req.Source)
	if cerr := tmp.Close(); cerr != nil && copyErr == nil {
		copyErr = cerr
	}
	if copyErr != nil {
		return Import{}, fmt.Errorf("import: spool: %w", copyErr)
	}

	zr, err := zip.OpenReader(tmpName)
	if err != nil {
		return Import{}, fmt.Errorf("%w: %v", ErrImportBadBundle, err)
	}
	defer zr.Close()

	// Validate manifest first so a malformed bundle fails fast,
	// before any DB writes.
	manifest, err := readImportManifest(zr)
	if err != nil {
		return Import{}, err
	}
	if manifest.Version != BundleVersion {
		return Import{}, fmt.Errorf("%w: bundle version %q != server %q",
			ErrImportBadBundle, manifest.Version, BundleVersion)
	}

	imp := Import{
		ID:             NewImportID(),
		Status:         ImportRunning,
		RequestedBy:    req.RequestedBy,
		StartedAt:      time.Now().UTC(),
		SourceFilename: req.Filename,
		SourceBytes:    bytesIn,
	}
	if err := s.store.InsertImport(ctx, imp); err != nil {
		return Import{}, err
	}

	// Run each section. Per-section error doesn't abort — we want
	// the operator to see partial progress (e.g. memories OK,
	// integrations failed), so we collect counts and a final
	// error string instead.
	var firstErr error
	noteErr := func(name string, err error) {
		if err == nil {
			return
		}
		s.log.Warn("import section failed",
			"import_id", imp.ID, "section", name, "err", err)
		if firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", name, err)
		}
	}

	for _, f := range zr.File {
		switch f.Name {
		case "memories.jsonl":
			if req.Memories {
				c, err := s.importMemories(ctx, f)
				imp.Counts.Memories = c
				noteErr("memories", err)
			}
		case "integrations.jsonl":
			if req.Integrations {
				c, err := s.importIntegrations(ctx, f)
				imp.Counts.Integrations = c
				noteErr("integrations", err)
			}
		case "custom_tasks.jsonl":
			if req.CustomTasks {
				c, err := s.importCustomTasks(ctx, f)
				imp.Counts.CustomTasks = c
				noteErr("custom_tasks", err)
			}
		}
	}

	if firstErr != nil {
		_ = s.store.MarkImportFailed(ctx, imp.ID, firstErr.Error(), imp.Counts)
		imp.Status = ImportFailed
		imp.Error = firstErr.Error()
	} else {
		_ = s.store.MarkImportSucceeded(ctx, imp.ID, imp.Counts)
		imp.Status = ImportSucceeded
	}
	finished := time.Now().UTC()
	imp.FinishedAt = &finished

	s.log.Info("import done",
		"import_id", imp.ID,
		"status", imp.Status,
		"memories_created", imp.Counts.Memories.Created,
		"memories_skipped", imp.Counts.Memories.Skipped,
		"integrations_created", imp.Counts.Integrations.Created,
		"custom_tasks_created", imp.Counts.CustomTasks.Created)

	if firstErr != nil {
		return imp, firstErr
	}
	return imp, nil
}

func readImportManifest(zr *zip.ReadCloser) (ExportManifest, error) {
	for _, f := range zr.File {
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				return ExportManifest{}, fmt.Errorf("%w: open manifest: %v", ErrImportBadBundle, err)
			}
			body, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return ExportManifest{}, fmt.Errorf("%w: read manifest: %v", ErrImportBadBundle, err)
			}
			var mf ExportManifest
			if err := json.Unmarshal(body, &mf); err != nil {
				return ExportManifest{}, fmt.Errorf("%w: parse manifest: %v", ErrImportBadBundle, err)
			}
			return mf, nil
		}
	}
	return ExportManifest{}, fmt.Errorf("%w: no manifest.json", ErrImportBadBundle)
}

// importedIntegrationKeyHash is what we stamp into integrations.api_key_hash
// for imported rows. It's a non-bcrypt sentinel — bcrypt.Compare will
// always fail against any plaintext, so the integration cannot
// authenticate until the operator explicitly rotates the key.
const importedIntegrationKeyHash = "imported:no-plaintext-key-rotate-before-use"

func (s *Service) importMemories(ctx context.Context, f *zip.File) (EntityCounts, error) {
	var c EntityCounts
	rc, err := f.Open()
	if err != nil {
		return c, err
	}
	defer rc.Close()

	dec := json.NewDecoder(rc)
	for {
		var row map[string]any
		if err := dec.Decode(&row); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			c.Failed++
			continue
		}
		id, _ := row["id"].(string)
		scope, _ := row["scope"].(string)
		scopeKey, _ := row["scope_key"].(string)
		text, _ := row["text"].(string)
		if id == "" || scope == "" || text == "" {
			c.Failed++
			continue
		}
		metaRaw, _ := json.Marshal(row["metadata"])
		if len(metaRaw) == 0 || string(metaRaw) == "null" {
			metaRaw = []byte("{}")
		}
		// embedding stays NULL; embedder marker invites re-embed.
		res, err := s.pool.Exec(ctx, `
			INSERT INTO memories (id, scope, scope_key, text, embedder, metadata, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'imported_v1', $5::jsonb, NOW(), NOW())
			ON CONFLICT (id) DO NOTHING`,
			id, scope, scopeKey, text, metaRaw)
		if err != nil {
			c.Failed++
			continue
		}
		if res.RowsAffected() == 0 {
			c.Skipped++
		} else {
			c.Created++
		}
	}
	return c, nil
}

func (s *Service) importIntegrations(ctx context.Context, f *zip.File) (EntityCounts, error) {
	var c EntityCounts
	rc, err := f.Open()
	if err != nil {
		return c, err
	}
	defer rc.Close()

	dec := json.NewDecoder(rc)
	for {
		var row map[string]any
		if err := dec.Decode(&row); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			c.Failed++
			continue
		}
		id, _ := row["id"].(string)
		name, _ := row["name"].(string)
		baseURL, _ := row["base_url"].(string)
		routePrefix, _ := row["route_prefix"].(string)
		ver, _ := row["version"].(string)
		// base_url is allowed to be empty for consumer-only
		// integrations (they don't proxy outbound requests).
		if id == "" || name == "" {
			c.Failed++
			continue
		}
		// route_prefix can legitimately be empty (consumer-only
		// integrations) — but the schema requires UNIQUE non-null,
		// so we synthesise the documented placeholder.
		if routePrefix == "" {
			routePrefix = "_consumer_" + id
		}
		scopesRaw, _ := json.Marshal(row["scopes"])
		if len(scopesRaw) == 0 || string(scopesRaw) == "null" {
			scopesRaw = []byte("[]")
		}
		isSystem, _ := row["is_system"].(bool)

		// Try insert; route_prefix collisions also count as skip.
		res, err := s.pool.Exec(ctx, `
			INSERT INTO integrations
				(id, name, base_url, route_prefix, api_key_hash,
				 scopes, version, enabled, health_status, created_at, is_system)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, FALSE, 'unknown', NOW(), $8)
			ON CONFLICT (id) DO NOTHING`,
			id, name, baseURL, routePrefix, importedIntegrationKeyHash,
			scopesRaw, nullIfEmpty(ver), isSystem)
		if err != nil {
			// Likely a UNIQUE violation on route_prefix. Treat as skip.
			if strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "unique constraint") {
				c.Skipped++
				continue
			}
			c.Failed++
			continue
		}
		if res.RowsAffected() == 0 {
			c.Skipped++
		} else {
			c.Created++
		}
	}
	return c, nil
}

func (s *Service) importCustomTasks(ctx context.Context, f *zip.File) (EntityCounts, error) {
	var c EntityCounts
	rc, err := f.Open()
	if err != nil {
		return c, err
	}
	defer rc.Close()

	dec := json.NewDecoder(rc)
	for {
		var row map[string]any
		if err := dec.Decode(&row); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			c.Failed++
			continue
		}
		id, _ := row["id"].(string)
		name, _ := row["name"].(string)
		cmd, _ := row["command"].(string)
		desc, _ := row["description"].(string)
		cwd, _ := row["cwd"].(string)
		if id == "" || name == "" || cmd == "" {
			c.Failed++
			continue
		}
		res, err := s.pool.Exec(ctx, `
			INSERT INTO custom_tasks (id, name, command, description, cwd, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			ON CONFLICT (id) DO NOTHING`,
			id, name, cmd, desc, cwd)
		if err != nil {
			c.Failed++
			continue
		}
		if res.RowsAffected() == 0 {
			c.Skipped++
		} else {
			c.Created++
		}
	}
	return c, nil
}

// ─── reads ────────────────────────────────────────────────────────

func (s *Service) GetImport(ctx context.Context, id string) (Import, error) {
	return s.store.GetImport(ctx, id)
}

func (s *Service) ListImports(ctx context.Context, limit int) ([]Import, error) {
	return s.store.ListImports(ctx, limit)
}
