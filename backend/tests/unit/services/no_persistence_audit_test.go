package services_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestKat_NoPersistenceAudit is a structural guardrail against accidentally
// re-introducing persistence into the Kat code path. It walks the Kat-owned
// source files with go/parser and asserts:
//   - no import of gorm.io/gorm or its driver packages
//   - no import of the project's repositories package
//   - no string literal that looks like a CREATE TABLE / INSERT INTO targeting
//     a kat_* table
//
// See .kiro/specs/kat-live-notes/requirements.md Requirement 1 (Ephemerality,
// No Persistence).
func TestKat_NoPersistenceAudit(t *testing.T) {
	// repoRoot resolves to backend/. The test runs from
	// backend/tests/unit/services so we step up three levels.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(cwd, "..", "..", ".."))

	katFiles := []string{
		"internal/application/services/kat_notes_service.go",
		"internal/application/services/kat_note_value.go",
		"internal/application/services/kat_speaker_labeller.go",
		"internal/infrastructure/foundry/client.go",
		"internal/infrastructure/foundry/note_generator.go",
		"internal/infrastructure/foundry/prompt.go",
		"internal/infrastructure/foundry/errors.go",
		"internal/interfaces/http/handlers/kat_handler.go",
	}

	bannedImports := []string{
		"gorm.io/gorm",
		"gorm.io/driver",
		"github.com/rekall/backend/internal/infrastructure/repositories",
	}

	// Match a CREATE TABLE / INSERT INTO / DROP TABLE referencing a kat_*
	// identifier. Case-insensitive. Requires a word boundary so "kat_v1"
	// (the prompt version literal) doesn't trip the matcher.
	bannedSQL := regexp.MustCompile(`(?i)(create\s+table|insert\s+into|drop\s+table|update\s+kat|delete\s+from\s+kat)\s+kat_\w+`)

	fset := token.NewFileSet()
	for _, rel := range katFiles {
		full := filepath.Join(repoRoot, rel)
		t.Run(rel, func(t *testing.T) {
			src, err := os.ReadFile(full)
			if err != nil {
				t.Fatalf("read %s: %v", full, err)
			}
			file, err := parser.ParseFile(fset, full, src, parser.ImportsOnly|parser.ParseComments)
			if err != nil {
				t.Fatalf("parse %s: %v", full, err)
			}
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				for _, banned := range bannedImports {
					if strings.HasPrefix(path, banned) {
						t.Errorf("%s imports %q — Kat must not reach the persistence layer", rel, path)
					}
				}
			}

			// Re-parse with full bodies to scan string literals / comments
			// for the banned SQL patterns.
			full, err := parser.ParseFile(fset, filepath.Join(repoRoot, rel), src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse-full %s: %v", rel, err)
			}
			_ = full
			if loc := bannedSQL.FindIndex(src); loc != nil {
				t.Errorf("%s contains a banned SQL pattern: %q", rel, src[loc[0]:loc[1]])
			}
		})
	}
}
