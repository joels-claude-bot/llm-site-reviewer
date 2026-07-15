package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/joels-claude-bot/llm-site-reviewer/internal/finding"
)

type User struct {
	Name   string
	Age    int
	Active bool
}

func main() {
	u := User{Name: "Alice", Age: 30, Active: true}
	jsonData, err := json.Marshal(u)
	if err != nil {
		log.Fatalf("failed to marshal json %v", err)
	}
	fmt.Println("got some json:\n", string(jsonData))
	blankUser := User{}
	ref := &blankUser
	schema := jsonschema.Reflect(ref)

	// 2. Marshal the schema to pretty-printed JSON
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Failed to generate schema: %s", err)
	}

	// 3. Print it out
	fmt.Println(string(schemaJSON))
	// fmt.Println("printing user shema:\n", result)

	myStr := "hi there"
	var myFinding finding.Finding
	fmt.Println("finding category is: ", myFinding.Category)

	// 1. Create the reader (our "stream")
	myReader := strings.NewReader(myStr)

	// 2. Wrap it in a Scanner. By default, it splits by line,
	// but we can tell it to split by individual characters (runes).
	scanner := bufio.NewScanner(myReader)
	scanner.Split(bufio.ScanRunes)

	// 3. Loop! scanner.Scan() is your "while yield" check.
	// It advances to the next character and returns 'false' when it hits the end.
	for scanner.Scan() {
		char := scanner.Text()
		fmt.Println("got a char: ", char)
	}
}
