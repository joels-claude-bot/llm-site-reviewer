// Package codemap builds the architecture map from //arch: tags in doc comments.
// Because it parses the real AST, every entry points at a symbol that exists: a
// tag cannot outlive the code it sits on, so the map cannot drift structurally.
//
// Tag a package, function, or type by putting a directive line in its doc
// comment, for example:
//
//	//arch:pure
//	func Parse(content []byte) (Fixture, error) { ... }
//
// The same engine that finds tagged symbols here is what the reviewer's
// STALE_REF pass will use to tell whether a documented function exists.
package codemap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// Entry is one tagged symbol: its role and where it lives in the source.
type Entry struct {
	Role     string // from //arch:<role>
	Pkg      string
	Name     string
	Kind     string // "package", "func", or "type"
	Synopsis string
	File     string
	Line     int
}

// Extract parses every non-test .go file under roots and returns one Entry per
// //arch:-tagged symbol. Only this reads disk; everything else is pure.
//
//arch:io
func Extract(roots ...string) ([]Entry, error) {
	fset := token.NewFileSet()
	var entries []Entry

	for _, root := range roots {
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			entries = append(entries, fileEntries(fset, file)...)
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return entries, nil
}

// fileEntries pulls the tagged package, funcs, and types out of one parsed file.
func fileEntries(fset *token.FileSet, file *ast.File) []Entry {
	pkg := file.Name.Name
	var entries []Entry

	if role, ok := tagOf(file.Doc); ok {
		entries = append(entries, entryAt(fset, file.Package, role, pkg, pkg, "package", file.Doc))
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if role, ok := tagOf(d.Doc); ok {
				entries = append(entries, entryAt(fset, d.Pos(), role, pkg, d.Name.Name, "func", d.Doc))
			}
		case *ast.GenDecl:
			if role, ok := tagOf(d.Doc); ok {
				if name := typeName(d); name != "" {
					entries = append(entries, entryAt(fset, d.Pos(), role, pkg, name, "type", d.Doc))
				}
			}
		}
	}
	return entries
}

func entryAt(fset *token.FileSet, pos token.Pos, role, pkg, name, kind string, doc *ast.CommentGroup) Entry {
	at := fset.Position(pos)
	return Entry{
		Role:     role,
		Pkg:      pkg,
		Name:     name,
		Kind:     kind,
		Synopsis: synopsis(doc),
		File:     filepath.ToSlash(at.Filename),
		Line:     at.Line,
	}
}

// tagOf returns the role from the first //arch:<role> directive in doc.
func tagOf(doc *ast.CommentGroup) (string, bool) {
	if doc == nil {
		return "", false
	}
	for _, comment := range doc.List {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(comment.Text), "//arch:"); ok {
			if role := strings.TrimSpace(rest); role != "" {
				return role, true
			}
		}
	}
	return "", false
}

// synopsis is the doc comment's first sentence, on one line. Text() has already
// stripped the //arch: line, so it comes back clean.
func synopsis(doc *ast.CommentGroup) string {
	text := strings.Join(strings.Fields(doc.Text()), " ")
	if dot := strings.Index(text, ". "); dot >= 0 {
		text = text[:dot+1]
	}
	return text
}

func typeName(d *ast.GenDecl) string {
	for _, spec := range d.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			return ts.Name.Name
		}
	}
	return ""
}
