package scaffold

import (
	"cmp"
	"slices"
	"strings"
	"unicode"
)

var stdlibModules = map[string]bool{"os": true, "pathlib": true, "sys": true, "typing": true}

type importLine struct {
	text    string
	module  string
	member  string
	section int
	from    bool
}

func parseImport(line string, firstParty []string) importLine {
	imp := importLine{text: line}
	if rest, ok := strings.CutPrefix(line, "from "); ok {
		imp.from = true
		imp.module, imp.member, _ = strings.Cut(rest, " import ")
	} else {
		imp.module = strings.TrimPrefix(line, "import ")
	}
	root, _, _ := strings.Cut(imp.module, ".")
	switch {
	case strings.HasPrefix(imp.module, "."):
		imp.section = 3
	case slices.Contains(firstParty, root):
		imp.section = 2
	case stdlibModules[root]:
		imp.section = 0
	default:
		imp.section = 1
	}
	return imp
}

func memberRank(name string) int {
	switch {
	case strings.ToUpper(name) == name:
		return 0
	case unicode.IsUpper(rune(name[0])):
		return 1
	default:
		return 2
	}
}

func orderImports(lines []string, firstParty []string) []string {
	parsed := make([]importLine, 0, len(lines))
	seen := map[string]bool{}
	for _, line := range lines {
		if !seen[line] {
			seen[line] = true
			parsed = append(parsed, parseImport(line, firstParty))
		}
	}
	slices.SortFunc(parsed, func(a, b importLine) int {
		if c := cmp.Compare(a.section, b.section); c != 0 {
			return c
		}
		if a.from != b.from {
			if a.from {
				return 1
			}
			return -1
		}
		if c := cmp.Compare(strings.ToLower(a.module), strings.ToLower(b.module)); c != 0 {
			return c
		}
		if c := cmp.Compare(memberRank(a.member), memberRank(b.member)); c != 0 {
			return c
		}
		return cmp.Compare(strings.ToLower(a.member), strings.ToLower(b.member))
	})
	var out []string
	for i, imp := range parsed {
		if i > 0 && imp.section != parsed[i-1].section {
			out = append(out, "")
		}
		out = append(out, imp.text)
	}
	return out
}

func (p project) firstPartyModules() []string {
	modules := []string{p.cfg.ModuleName(), coreApp, accountsApp}
	if p.cfg.App().Ok() {
		modules = append(modules, p.cfg.App().String())
	}
	return modules
}
