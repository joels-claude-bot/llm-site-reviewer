// Package review defines the shared result contract and defect category catalogue.
//
//arch:types
package review

// Category is one defect type from the spec's category reference.
//
// TODO: the LLM's response is not yet constrained to these codes. Bake Catalog
// into the response schema as an enum, or validate the returned Category against
// it on parse, so a hallucinated code is rejected instead of stored as-is.
type Category string

// Category codes recognized by the reviewer.
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

// CategoryExample is the compact information included in the batch prompt.
type CategoryExample struct {
	Code    Category `json:"code"`
	Example string   `json:"example"`
}

// Catalog lists the supported categories and a short example for each.
var Catalog = []CategoryExample{
	{BrokenExternalLink, "external page returns 404"},
	{Soft404, `page returns 200 but title says "Page not found"`},
	{RenderedBroken, "Mermaid source appears instead of a diagram"},
	{DiagramSyntaxError, "diagram renders an error box"},
	{Truncation, "page cuts off mid-section"},
	{TextLegibility, "diagram text is too small to read"},
	{FlatColour, "page is a grey wall with no visual hierarchy"},
	{WeakHierarchy, "scan cannot tell what matters most"},
	{HighDensity, "huge table or unbroken prose block"},
	{AcronymUnexpanded, "GDS appears with no explanation"},
	{AssumedJargon, `"send it through the GDS" with no context`},
	{MissingWhatWhy, "page starts with commands before saying what they are for"},
	{MissingDoesNot, "page never states what it does not cover"},
	{Simpler, "complex prose that should be a table or short example"},
	{StaleRef, "docs name process_frame(), but the repo has no such function"},
	{WrongClaim, "docs say timeout is 30 seconds, code sets 60"},
	{ContradictsPage, "page A says default on, page B says default off"},
	{InconsistentTerm, "same thing is called doc_build, build dir, and output/"},
	{BreaksStandard, "style guide says expand acronyms; page does not"},
	{Unverifiable, "claim cannot be checked from available context"},
	{ImageMismatch, "text asks for June 2026; screenshot shows July 2026"},
}
