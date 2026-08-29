package review

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Example_buildMessages() {
	dir, err := os.MkdirTemp("", "review-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	writeExampleFile(dir, "a.md", "# A\n")
	writeExampleFile(dir, "b.txt", "ignored\n")
	writeExampleFile(dir, "c.png", "png bytes")
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0755); err != nil {
		panic(err)
	}
	writeExampleFile(filepath.Join(dir, "nested"), "deep.md", "# Deep\n")

	entries, err := buildMessages([]string{dir}, true)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		fmt.Printf("%d %s text=%t image=%t\n", entry.Id, entry.FileName, entry.TextContent != "", entry.ImageContent != nil)
	}

	// Output:
	// 0 a.md text=true image=false
	// 1 c.png text=false image=true
	// 2 deep.md text=true image=false
}

func ExampleResolveMarkdownImages() {
	dir, err := os.MkdirTemp("", "review-images-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	writeExampleFile(dir, "route.svg", "<svg>route</svg>")
	mdPath := filepath.Join(dir, "page.md")

	images, err := ResolveMarkdownImages([]byte("![Route](route.svg)\n"), mdPath, dir)
	if err != nil {
		panic(err)
	}
	for _, image := range images {
		fmt.Printf("%s -> %s remote=%t bytes=%d\n", image.Ref, filepath.Base(image.Resolved), image.IsRemote, len(image.Content))
	}

	// Output:
	// route.svg -> route.svg remote=false bytes=16
}

func Example_buildReviewPayload() {
	page := docPage{
		Path:    "research.md",
		Content: "# Research\nSee spec.md.\n",
		Images: []docImage{
			{Path: "route.svg", Summary: "route takes 8 hours"},
		},
		DependencyPaths: []string{"spec.md"},
	}
	byPath := map[string]docPage{
		"research.md": page,
		"spec.md":     {Path: "spec.md", Content: "# Spec\nHotel must be within 30 minutes.\n"},
	}

	payload, err := buildReviewPayload(page, byPath)
	if err != nil {
		panic(err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	// Output:
	// {
	//   "target": {
	//     "path": "research.md",
	//     "content": "# Research\nSee spec.md.\n"
	//   },
	//   "images": [
	//     {
	//       "path": "route.svg",
	//       "content": "route takes 8 hours"
	//     }
	//   ],
	//   "dependencies": [
	//     {
	//       "path": "spec.md",
	//       "content": "# Spec\nHotel must be within 30 minutes.\n"
	//     }
	//   ]
	// }
}

func Example_validatePaths() {
	byPath := map[string]docPage{
		"index.md":    {Path: "index.md"},
		"spec.md":     {Path: "spec.md"},
		"research.md": {Path: "research.md"},
	}

	paths := validatePaths("research.md", []string{"spec.md", "index.md"}, byPath)
	fmt.Println(paths)

	// Output:
	// [spec.md index.md]
}

func writeExampleFile(dir, name, content string) {
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		panic(err)
	}
}
