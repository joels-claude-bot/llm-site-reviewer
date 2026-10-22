// Package review defines the shared JSON result contract for review engines.
// It contains no model clients, filesystem operations, or review pipeline.
package review

import (
	"github.com/invopop/jsonschema"
)

// Issue describes a defect found in a reviewed page.
type Issue struct {
	Category Category `json:"category"`
	Notes    string   `json:"notes"`
}

// PageResult identifies the reviewed input and its findings.
type PageResult struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Success bool    `json:"success"`
	Errors  []Issue `json:"errors"`
}

// Response is the result returned by either review engine.
type Response struct {
	Results []PageResult `json:"results"`
}

// ResponseSchema uses inline definitions for provider compatibility.
func ResponseSchema() *jsonschema.Schema {
	return (&jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}).Reflect(&Response{})
}
