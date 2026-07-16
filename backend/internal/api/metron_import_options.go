package api

import "strings"

func defaultMetronImportOptions() MetronImportOptions {
	return MetronImportOptions{Mode: "quick"}
}

func resolveMetronImportOptions(options MetronImportOptions) MetronImportOptions {
	force := options.Force
	switch options.Mode {
	case "full":
		return MetronImportOptions{Mode: "full", FullData: normalizeMetronFullData(options.FullData), Force: force}
	default:
		next := defaultMetronImportOptions()
		next.Force = force
		return next
	}
}

func normalizeMetronFullData(values []string) []string {
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(strings.ToLower(value))
		switch value {
		case "comic", "comics", "issue", "issues":
			seen["comics"] = true
		case "series":
			seen["series"] = true
		case "arc", "arcs":
			seen["arcs"] = true
		case "character", "characters":
			seen["characters"] = true
		}
	}
	if len(values) == 0 {
		values = []string{"comics", "series", "arcs", "characters"}
	}
	for _, value := range values {
		add(value)
	}
	if seen["series"] || seen["arcs"] || seen["characters"] {
		seen["comics"] = true
	}
	ordered := []string{}
	for _, value := range []string{"comics", "series", "arcs", "characters"} {
		if seen[value] {
			ordered = append(ordered, value)
		}
	}
	return ordered
}

func (o MetronImportOptions) includesFullData(value string) bool {
	o = resolveMetronImportOptions(o)
	if o.Mode != "full" {
		return false
	}
	for _, item := range o.FullData {
		if item == value {
			return true
		}
	}
	return false
}

func (o MetronImportOptions) includesComics() bool {
	return o.includesFullData("comics")
}

func (o MetronImportOptions) includesSeries() bool {
	return o.includesFullData("series")
}

func (o MetronImportOptions) includesArcs() bool {
	return o.includesFullData("arcs")
}

func (o MetronImportOptions) includesCharacters() bool {
	return o.includesFullData("characters")
}

func (o MetronImportOptions) needsIssueDetail() bool {
	return o.includesComics() || o.includesSeries() || o.includesArcs() || o.includesCharacters()
}
