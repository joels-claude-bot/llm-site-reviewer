// Package finding defines a reported defect: its category, and whether that
// category can block a build or is only reported.
package finding

// Result is what happens when a finding is raised.
type Result string

const (
	// Blocking findings are provable, so they can fail a build.
	Blocking Result = "blocking"
	// Report findings need judgment, so they are shown but never fail a build.
	Report Result = "report"
)

// Category is one defect type from the spec's category reference.
type Category string

// The full set, grouped as in the reference. Keep this list in step with
// docs-site/docs/spec/reference.md; the corpus test checks every fixture
// against it, so a typo or a renamed code fails loudly.
const (
	// Links.
	BrokenInternalLink Category = "BROKEN_INTERNAL_LINK"
	BrokenAnchor       Category = "BROKEN_ANCHOR"
	MissingImage       Category = "MISSING_IMAGE"
	BrokenExternalLink Category = "BROKEN_EXTERNAL_LINK"
	Soft404            Category = "SOFT_404"
	OrphanedPage       Category = "ORPHANED_PAGE"

	// Rendering.
	RenderedBroken     Category = "RENDERED_BROKEN"
	DiagramSyntaxError Category = "DIAGRAM_SYNTAX_ERROR"
	Truncation         Category = "TRUNCATION"
	TextLegibility     Category = "TEXT_LEGIBILITY"
	FlatColour         Category = "FLAT_COLOUR"
	WeakHierarchy      Category = "WEAK_HIERARCHY"
	HighDensity        Category = "HIGH_DENSITY"

	// Clarity.
	AcronymUnexpanded Category = "ACRONYM_UNEXPANDED"
	AssumedJargon     Category = "ASSUMED_JARGON"
	MissingWhatWhy    Category = "MISSING_WHAT_WHY"
	MissingDoesNot    Category = "MISSING_DOES_NOT"
	Simpler           Category = "SIMPLER"

	// Mismatch.
	StaleRef         Category = "STALE_REF"
	WrongClaim       Category = "WRONG_CLAIM"
	ContradictsPage  Category = "CONTRADICTS_PAGE"
	InconsistentTerm Category = "INCONSISTENT_TERM"
	BreaksStandard   Category = "BREAKS_STANDARD"
	Unverifiable     Category = "UNVERIFIABLE"
	ImageMismatch    Category = "IMAGE_MISMATCH"
)

var known = map[Category]struct{}{
	BrokenInternalLink: {}, BrokenAnchor: {}, MissingImage: {}, BrokenExternalLink: {}, Soft404: {}, OrphanedPage: {},
	RenderedBroken: {}, DiagramSyntaxError: {}, Truncation: {}, TextLegibility: {}, FlatColour: {}, WeakHierarchy: {}, HighDensity: {},
	AcronymUnexpanded: {}, AssumedJargon: {}, MissingWhatWhy: {}, MissingDoesNot: {}, Simpler: {},
	StaleRef: {}, WrongClaim: {}, ContradictsPage: {}, InconsistentTerm: {}, BreaksStandard: {}, Unverifiable: {}, ImageMismatch: {},
}

// Known reports whether c is a category from the reference.
func (c Category) Known() bool {
	_, ok := known[c]
	return ok
}

// Finding is one reported defect: its category, where on the page it sits,
// and what should happen when it is raised.
type Finding struct {
	Category Category
	Where    string
	Result   Result
}
