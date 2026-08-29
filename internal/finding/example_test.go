package finding

import "fmt"

func ExampleCatalog() {
	for _, meta := range Catalog[:3] {
		fmt.Printf("%s: %s (%s)\n", meta.Code, meta.Example, meta.How)
	}

	// Output:
	// BROKEN_EXTERNAL_LINK: external page returns 404 (Mechanical)
	// SOFT_404: page returns 200 but title says "Page not found" (Cognitive Text Review)
	// RENDERED_BROKEN: Mermaid source appears instead of a diagram (Cognitive Screenshot Review)
}
