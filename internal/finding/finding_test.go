package finding

import "testing"

func TestCategoryKnown(t *testing.T) {
	if !BrokenInternalLink.Known() {
		t.Error("a category from the reference should be Known")
	}
	if Category("NOT_A_REAL_CODE").Known() {
		t.Error("an invented category should not be Known")
	}
}

// Guard the catalog's invariants: every entry is a unique category with a
// non-empty group/example and a real result. A slip here fails loudly instead of
// generating a broken reference page.
func TestCatalogIsWellFormed(t *testing.T) {
	seen := map[Category]bool{}
	for _, meta := range Catalog {
		if seen[meta.Code] {
			t.Errorf("duplicate category %q in Catalog", meta.Code)
		}
		seen[meta.Code] = true

		if meta.Group == "" || meta.Example == "" {
			t.Errorf("category %q has an empty group or example", meta.Code)
		}
		if !meta.Code.Known() {
			t.Errorf("category %q is in Catalog but not Known()", meta.Code)
		}
	}
}

// known is derived from Catalog, so the two must have the same size.
func TestKnownMatchesCatalog(t *testing.T) {
	if len(known) != len(Catalog) {
		t.Errorf("known has %d entries, Catalog has %d", len(known), len(Catalog))
	}
}
