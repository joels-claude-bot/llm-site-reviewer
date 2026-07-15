// Package finding defines a reported defect (its category, where it sits, and
// its result) and holds Catalog, the source of truth that the generated category
// reference and the known-category set both derive from.
//
//arch:types
package finding

// Finding is one reported defect: its category, where on the page it sits, and
// what should happen when it is raised.
type Finding struct {
	Category Category
	Where    string
	Result   Result
}

// Category is one defect type from the spec's category reference.
type Category string

// The full set, grouped as in the reference. Each also has a row in Catalog
// (below), the source of truth the docs page and known set derive from, so a
// typo or renamed code fails loudly in the corpus test instead of drifting.
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

// Known reports whether c is a category from the reference.
func (c Category) Known() bool {
	_, ok := known[c]
	return ok
}

// known is derived from Catalog (below) so there's one list, not two: a category
// is Known once it's in Catalog.
var known = func() map[Category]struct{} {
	set := make(map[Category]struct{}, len(Catalog))
	for _, meta := range Catalog {
		set[meta.Code] = struct{}{}
	}
	return set
}()

// Result is what happens when a finding is raised. For now every category is
// Blocking; Report is kept for the corpus and future non-blocking findings.
type Result string

const (
	// Blocking findings are provable, so they can fail a build.
	Blocking Result = "blocking"
	// Report findings need judgment, so they are shown but never fail a build.
	Report Result = "report"
)

// How says which engine decides a category: a tool, or a cognitive (LLM) pass
// over the text or a screenshot. A model is only spent where a tool cannot prove
// the answer.
type How string

const (
	Mechanical                How = "Mechanical"
	CognitiveTextReview       How = "Cognitive Text Review"
	CognitiveScreenshotReview How = "Cognitive Screenshot Review"
)

// HowInfo pairs a How with the one-line meaning shown in the generated table, so
// the "How checked" column is explained there rather than just labelled.
type HowInfo struct {
	How  How
	Desc string
}

// HowKinds is every How in reference order, each with its description. The
// generated page renders it as the "How checked" glossary.
var HowKinds = []HowInfo{
	{Mechanical, "A tool proves the answer exactly; no model is involved."},
	{CognitiveTextReview, "A text model reads the prose and judges what needs judgment."},
	{CognitiveScreenshotReview, "A vision model looks at a screenshot of the rendered page."},
}

// Meta is everything the reference records about one category: its group, a
// human example, and how it is checked.
type Meta struct {
	Code    Category
	Group   string
	Example string
	How     How
}

// Catalog is every category in reference order. It reuses the typed constants
// above, so the code API and the docs can't end up with different codes.
// cmd/refdocs generates docs-site/docs/spec/categories.md from Catalog, HowKinds,
// and LookAlikes, so edit here and regenerate rather than touching that page.
var Catalog = []Meta{
	// Links.
	{BrokenExternalLink, "Links", "external page returns 404", Mechanical},
	{Soft404, "Links", `page returns 200 but title says "Page not found"`, CognitiveTextReview},

	// Rendering.
	{RenderedBroken, "Rendering", "Mermaid source appears instead of a diagram", CognitiveScreenshotReview},
	{DiagramSyntaxError, "Rendering", "diagram renders an error box", CognitiveScreenshotReview},
	{Truncation, "Rendering", "page cuts off mid-section", CognitiveScreenshotReview},
	{TextLegibility, "Rendering", "diagram text is too small to read", CognitiveScreenshotReview},
	{FlatColour, "Rendering", "page is a grey wall with no visual hierarchy", CognitiveScreenshotReview},
	{WeakHierarchy, "Rendering", "scan cannot tell what matters most", CognitiveScreenshotReview},
	{HighDensity, "Rendering", "huge table or unbroken prose block", CognitiveScreenshotReview},

	// Clarity.
	{AcronymUnexpanded, "Clarity", "`GDS` appears with no explanation", CognitiveTextReview},
	{AssumedJargon, "Clarity", `"send it through the GDS" with no context`, CognitiveTextReview},
	{MissingWhatWhy, "Clarity", "page starts with commands before saying what they are for", CognitiveTextReview},
	{MissingDoesNot, "Clarity", "page never states what it does not cover", CognitiveTextReview},
	{Simpler, "Clarity", "complex prose that should be a table or short example", CognitiveTextReview},

	// Mismatch.
	{StaleRef, "Mismatch", "docs name `process_frame()`, but the repo has no such function", Mechanical},
	{WrongClaim, "Mismatch", "docs say timeout is 30 seconds, code sets 60", CognitiveTextReview},
	{ContradictsPage, "Mismatch", "page A says default on, page B says default off", CognitiveTextReview},
	{InconsistentTerm, "Mismatch", "same thing is called `doc_build`, `build dir`, and `output/`", CognitiveTextReview},
	{BreaksStandard, "Mismatch", "style guide says expand acronyms; page does not", CognitiveTextReview},
	{Unverifiable, "Mismatch", "claim cannot be checked from available context", CognitiveTextReview},
	{ImageMismatch, "Mismatch", "text asks for June 2026; screenshot shows July 2026", CognitiveScreenshotReview},
}

// LookAlike is something that looks like a defect but isn't. Listed so the
// reviewer (and its tests) leave them alone.
type LookAlike struct {
	Looks string
	Why   string
}

// LookAlikes is the "not a defect" list on the reference page.
var LookAlikes = []LookAlike{
	{"`https://example.com` inside a code block", "It is example code, not a docs link"},
	{"`ntfy.sh/your-topic` inside a shell snippet", "Placeholder value for a user to replace"},
	{"`# TODO` in an explicit stub section", "Known incomplete work, not a new finding"},
	{"acronym defined in a glossary", "The reader already has the context"},
	{"intentional 404 page in an error-handling guide", "The 404 is the example"},
}
