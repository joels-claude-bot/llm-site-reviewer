// Package corpus loads the fixture pages the reviewer is tested against. Each
// fixture is a small page that plants one defect, and the result we expect from
// it lives in the same file's frontmatter, so the two stay in sync.
//
// The work splits into three pieces, each with one job:
//
//	Parse    decode one fixture's bytes         (pure, no IO)
//	Validate check one fixture against the rules (pure, no IO)
//	Load     read and parse every fixture        (reads files)
//
// Parse and Validate take a value and return a value, so they are tested with
// literal strings. Load is the only part that reads the disk.
package corpus

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
)

// Fixture is one corpus page plus the result we expect from it.
type Fixture struct {
	Path     string            // path relative to the corpus root, e.g. "links/broken-internal.md"
	Category string            // the containing directory, e.g. "links"
	Title    string            // from frontmatter
	Expect   []finding.Finding // empty for a look-alike, which must produce nothing
	Note     string            // human explanation of the planted defect
	Body     string            // markdown after the frontmatter
}

// LookAlike reports whether this fixture should produce no findings.
func (f Fixture) LookAlike() bool { return len(f.Expect) == 0 }

// matter is the raw frontmatter shape, decoded straight from YAML.
type matter struct {
	Title  string `yaml:"title"`
	Note   string `yaml:"note"`
	Expect []struct {
		Category string `yaml:"category"`
		Where    string `yaml:"where"`
		Result   string `yaml:"result"`
	} `yaml:"expect"`
}

// Parse decodes one fixture's bytes into a Fixture. Pure, so tested with literal
// strings. Load fills in Path and Category; Validate does the rule-checking.
//
//arch:pure
func Parse(content []byte) (Fixture, error) {
	var m matter
	body, err := frontmatter.Parse(bytes.NewReader(content), &m)
	if err != nil {
		return Fixture{}, fmt.Errorf("frontmatter: %w", err)
	}

	fixture := Fixture{
		Title: m.Title,
		Note:  m.Note,
		Body:  string(bytes.TrimSpace(body)),
	}
	for _, e := range m.Expect {
		fixture.Expect = append(fixture.Expect, finding.Finding{
			Category: finding.Category(e.Category),
			Where:    e.Where,
			Result:   finding.Result(e.Result),
		})
	}
	return fixture, nil
}

// Validate returns the first corpus rule a fixture breaks, or nil. The rules: it
// needs a title; every expectation needs a known category, a valid result, and a
// 'where'; and only ok-* files may expect nothing.
//
//arch:pure
func Validate(fixture Fixture) error {
	if fixture.Title == "" {
		return fmt.Errorf("missing frontmatter title")
	}
	for _, want := range fixture.Expect {
		switch {
		case !want.Category.Known():
			return fmt.Errorf("unknown category %q (not in the reference)", want.Category)
		case want.Result != finding.Blocking && want.Result != finding.Report:
			return fmt.Errorf("category %q has invalid result %q", want.Category, want.Result)
		case want.Where == "":
			return fmt.Errorf("category %q has no 'where' (a reader cannot locate it)", want.Category)
		}
	}

	markedLookAlike := strings.HasPrefix(filepath.Base(fixture.Path), "ok-")
	switch {
	case markedLookAlike && !fixture.LookAlike():
		return fmt.Errorf("ok-* fixture must expect nothing, got %d", len(fixture.Expect))
	case !markedLookAlike && fixture.LookAlike():
		return fmt.Errorf("fixture expects nothing; plant a defect or rename to ok-*")
	}
	return nil
}

// Load reads and parses every fixture under root. It does not validate, so
// callers can load without rule-checking.
//
//arch:io
func Load(root string) ([]Fixture, error) {
	paths, err := FixturePaths(root)
	if err != nil {
		return nil, err
	}

	fixtures := make([]Fixture, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fixture, err := Parse(content)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if rel, err := filepath.Rel(root, path); err == nil {
			fixture.Path = filepath.ToSlash(rel)
		}
		fixture.Category = filepath.Base(filepath.Dir(path))
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

// FixturePaths returns every category/*.md fixture under root, sorted. It skips
// index.md, which isn't a test case: it only links the other pages so an
// orphaned page shows up.
//
//arch:io
func FixturePaths(root string) ([]string, error) {
	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read corpus root: %w", err)
	}

	var paths []string
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(root, dir.Name(), "*.md"))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			if filepath.Base(match) != "index.md" {
				paths = append(paths, match)
			}
		}
	}
	sort.Strings(paths)
	return paths, nil
}
