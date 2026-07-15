package codemap

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// Text renders the entries grouped by role, one jump-to file:line per symbol.
// Pure, so tested with literal entries.
//
//arch:pure
func Text(entries []Entry) string {
	byRole := map[string][]Entry{}
	for _, entry := range entries {
		byRole[entry.Role] = append(byRole[entry.Role], entry)
	}

	roles := make([]string, 0, len(byRole))
	for role := range byRole {
		roles = append(roles, role)
	}
	sort.Strings(roles)

	var out strings.Builder
	for _, role := range roles {
		group := byRole[role]
		sort.Slice(group, func(i, j int) bool {
			if group[i].File != group[j].File {
				return group[i].File < group[j].File
			}
			return group[i].Line < group[j].Line
		})
		fmt.Fprintf(&out, "%s\n", strings.ToUpper(role))
		for _, entry := range group {
			symbol := entry.Pkg + "." + entry.Name
			if entry.Kind == "package" {
				symbol = path.Dir(entry.File) + "/"
			}
			fmt.Fprintf(&out, "  %-24s %s:%d\n      %s\n", symbol, entry.File, entry.Line, entry.Synopsis)
		}
		out.WriteString("\n")
	}
	return out.String()
}
