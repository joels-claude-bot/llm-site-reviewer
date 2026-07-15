package codemap

// Internal test (package codemap) so it can reach the unexported helpers.

import (
	"go/ast"
	"testing"
)

func comments(lines ...string) *ast.CommentGroup {
	group := &ast.CommentGroup{}
	for _, line := range lines {
		group.List = append(group.List, &ast.Comment{Text: line})
	}
	return group
}

func TestTagOf(t *testing.T) {
	if role, ok := tagOf(comments("// A function.", "//arch:pure")); !ok || role != "pure" {
		t.Errorf("tagOf = (%q, %v), want (pure, true)", role, ok)
	}
	if _, ok := tagOf(comments("// no tag here")); ok {
		t.Error("tagOf found a tag where there is none")
	}
	if _, ok := tagOf(nil); ok {
		t.Error("tagOf(nil) should report no tag")
	}
}

func TestSynopsis(t *testing.T) {
	got := synopsis(comments("// Parse decodes bytes. It does more too."))
	if got != "Parse decodes bytes." {
		t.Errorf("synopsis = %q, want first sentence only", got)
	}
}

func TestTypeName(t *testing.T) {
	withType := &ast.GenDecl{Specs: []ast.Spec{&ast.TypeSpec{Name: ast.NewIdent("Finding")}}}
	if got := typeName(withType); got != "Finding" {
		t.Errorf("typeName = %q, want Finding", got)
	}
	if got := typeName(&ast.GenDecl{}); got != "" {
		t.Errorf("typeName of a non-type decl = %q, want empty", got)
	}
}
