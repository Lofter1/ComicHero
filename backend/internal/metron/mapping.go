package metron

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func issueFromMap(raw map[string]any) Issue {
	series := object(raw, "series")
	publisher := object(raw, "publisher")
	if len(publisher) == 0 {
		publisher = object(series, "publisher")
	}

	return Issue{
		ID:           intValue(raw, "id"),
		ComicVineID:  intValue(raw, "cv_id", "comic_vine_id", "comicVineId"),
		Title:        firstString(raw, "title", "name"),
		StoryNames:   stringList(raw, "name", "story_names", "storyNames"),
		SeriesID:     firstInt(intValue(raw, "series_id"), intValue(series, "id")),
		Series:       fallbackString(firstString(raw, "series_name", "series"), firstString(series, "name")),
		SeriesYear:   firstInt(intValue(raw, "series_year", "year_began", "year"), intValue(series, "year_began", "year")),
		SeriesVolume: firstInt(intValue(raw, "series_volume"), intValue(series, "volume")),
		Issue:        firstString(raw, "number", "issue"),
		Number:       firstString(raw, "number", "issue"),
		Publisher:    fallbackString(firstString(raw, "publisher_name", "publisher"), firstString(publisher, "name")),
		CoverDate:    firstString(raw, "cover_date", "store_date", "date"),
		StoreDate:    firstString(raw, "store_date"),
		CoverImage:   firstString(raw, "image", "cover", "cover_image", "image_url"),
		Description:  firstString(raw, "desc", "description", "synopsis"),
		Modified:     firstString(raw, "modified"),
		Tags:         stringList(raw, "tags", "tag", "issue_type"),
		Arcs:         arcsFromMap(raw),
		Characters:   charactersFromMap(raw),
	}
}

func arcsFromMap(raw map[string]any) []MetronArc {
	values, ok := raw["arcs"].([]any)
	if !ok {
		return nil
	}

	arcs := make([]MetronArc, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			arc := arcFromMap(item)
			if arc.ID > 0 || arc.Name != "" {
				arcs = append(arcs, arc)
			}
		}
	}
	return arcs
}

func charactersFromMap(raw map[string]any) []MetronCharacter {
	values, ok := raw["characters"].([]any)
	if !ok {
		return nil
	}

	characters := make([]MetronCharacter, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			character := characterFromMap(item)
			if character.ID > 0 || character.Name != "" {
				characters = append(characters, character)
			}
		}
	}
	return characters
}

func characterFromMap(raw map[string]any) MetronCharacter {
	return MetronCharacter{
		ID:          intValue(raw, "id"),
		Name:        firstString(raw, "name"),
		Aliases:     stringList(raw, "alias", "aliases"),
		Description: firstString(raw, "desc", "description"),
		Image:       firstString(raw, "image"),
	}
}

func stringList(raw map[string]any, keys ...string) []string {
	seen := map[string]bool{}
	var values []string
	for _, key := range keys {
		switch rawValue := raw[key].(type) {
		case string:
			value := strings.TrimSpace(rawValue)
			if value != "" && !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		case []string:
			for _, item := range rawValue {
				value := strings.TrimSpace(item)
				if value == "" || seen[value] {
					continue
				}
				seen[value] = true
				values = append(values, value)
			}
		case []any:
			for _, item := range rawValue {
				value := stringValue(item)
				if value == "" || seen[value] {
					continue
				}
				seen[value] = true
				values = append(values, value)
			}
		case map[string]any:
			value := stringValue(rawValue)
			if value != "" && !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		}
	}
	return values
}

func mergeStringLists(lists ...[]string) []string {
	seen := map[string]bool{}
	values := []string{}
	for _, list := range lists {
		for _, item := range list {
			value := strings.TrimSpace(item)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			values = append(values, value)
		}
	}
	return values
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return strings.TrimSpace(firstString(typed, "name", "title", "label", "value", "slug"))
	default:
		return ""
	}
}

func cloneMap(raw map[string]any) map[string]any {
	next := make(map[string]any, len(raw))
	for key, value := range raw {
		next[key] = value
	}
	return next
}

func firstInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func readingListFromMap(raw map[string]any) ReadingList {
	user := object(raw, "user")
	list := ReadingList{
		ID:                intValue(raw, "id"),
		Name:              firstString(raw, "name", "title"),
		Slug:              firstString(raw, "slug"),
		User:              MetronUser{ID: intValue(user, "id"), Username: firstString(user, "username", "name")},
		ListType:          firstString(raw, "list_type", "listType", "type"),
		IsPrivate:         boolValue(raw, "is_private", "isPrivate"),
		AttributionSource: firstString(raw, "attribution_source", "attributionSource"),
		AttributionURL:    firstString(raw, "attribution_url", "attributionUrl"),
		AverageRating:     floatValue(raw, "average_rating", "averageRating"),
		RatingCount:       intValue(raw, "rating_count", "ratingCount"),
		Modified:          firstString(raw, "modified"),
		Image:             firstString(raw, "image"),
		ItemsURL:          firstString(raw, "items_url", "itemsUrl"),
		ResourceURL:       firstString(raw, "resource_url", "resourceUrl"),
		Description:       firstString(raw, "desc", "description", "summary"),
	}

	for _, key := range []string{"issues", "issue", "comics"} {
		values, ok := raw[key].([]any)
		if !ok {
			continue
		}
		for _, value := range values {
			if item, ok := value.(map[string]any); ok {
				list.Issues = append(list.Issues, issueFromMap(item))
			}
		}
	}
	return list
}

func floatValue(raw map[string]any, keys ...string) float64 {
	for _, key := range keys {
		switch value := raw[key].(type) {
		case float64:
			return value
		case float32:
			return float64(value)
		case int:
			return float64(value)
		case int64:
			return float64(value)
		case json.Number:
			parsed, _ := value.Float64()
			return parsed
		case string:
			parsed, _ := strconv.ParseFloat(value, 64)
			return parsed
		}
	}
	return 0
}

func boolValue(raw map[string]any, keys ...string) bool {
	for _, key := range keys {
		switch value := raw[key].(type) {
		case bool:
			return value
		case float64:
			return value != 0
		case string:
			parsed, _ := strconv.ParseBool(value)
			return parsed
		}
	}
	return false
}

func arcFromMap(raw map[string]any) MetronArc {
	arc := MetronArc{
		ID:          intValue(raw, "id"),
		Name:        firstString(raw, "name", "title"),
		Description: firstString(raw, "desc", "description", "summary"),
		Image:       firstString(raw, "image"),
		Modified:    firstString(raw, "modified"),
	}

	for _, key := range []string{"issues", "issue", "comics"} {
		values, ok := raw[key].([]any)
		if !ok {
			continue
		}
		for _, value := range values {
			if item, ok := value.(map[string]any); ok {
				arc.Issues = append(arc.Issues, issueFromMap(item))
			}
		}
	}
	return arc
}

func seriesFromMap(raw map[string]any) Series {
	publisher := object(raw, "publisher")

	return Series{
		ID:          intValue(raw, "id"),
		Name:        firstString(raw, "name", "series"),
		Publisher:   fallbackString(firstString(raw, "publisher_name", "publisher"), firstString(publisher, "name")),
		Volume:      intValue(raw, "volume"),
		YearBegan:   intValue(raw, "year_began"),
		YearEnd:     intValue(raw, "year_end"),
		IssueCount:  intValue(raw, "issue_count"),
		Description: firstString(raw, "desc", "description"),
	}
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed
		case map[string]any:
			if name := firstString(typed, "url", "original_url", "image", "thumb_url", "name", "title"); name != "" {
				return name
			}
		}
	}
	return ""
}

func fallbackString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func object(raw map[string]any, key string) map[string]any {
	value, ok := raw[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

func intValue(raw map[string]any, keys ...string) int {
	for _, key := range keys {
		switch value := raw[key].(type) {
		case float64:
			return int(value)
		case int:
			return value
		case string:
			var parsed int
			if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func headerInt(header http.Header, key string) int {
	value, err := strconv.Atoi(header.Get(key))
	if err != nil {
		return 0
	}
	return value
}

func headerInt64(header http.Header, key string) int64 {
	value, err := strconv.ParseInt(header.Get(key), 10, 64)
	if err != nil {
		return 0
	}
	return value
}
