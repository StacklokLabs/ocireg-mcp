package mcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

// Pagination defaults and limits.
const (
	DefaultPageSize = 100
	MaxPageSize     = 1000
)

// Sort order constants for list_tags.
const (
	SortAlphabetical     = "alphabetical"
	SortAlphabeticalDesc = "alphabetical-desc"
	SortSemver           = "semver"
	SortSemverDesc       = "semver-desc"
)

// isValidSortOrder returns true if the given order is a recognized sort order.
func isValidSortOrder(order string) bool {
	switch order {
	case SortAlphabetical, SortAlphabeticalDesc, SortSemver, SortSemverDesc:
		return true
	default:
		return false
	}
}

// listTagsCursor represents the pagination state for list_tags.
type listTagsCursor struct {
	Offset int    `json:"o"`
	Sort   string `json:"s"`
}

// encodeCursor encodes pagination state into an opaque cursor string.
func encodeCursor(offset int, sortOrder string) string {
	c := listTagsCursor{Offset: offset, Sort: sortOrder}
	data, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(data)
}

// decodeCursor decodes an opaque cursor string into pagination state.
func decodeCursor(cursorStr string) (listTagsCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(cursorStr)
	if err != nil {
		return listTagsCursor{}, fmt.Errorf("invalid cursor: %w", err)
	}

	var c listTagsCursor
	if err := json.Unmarshal(data, &c); err != nil {
		return listTagsCursor{}, fmt.Errorf("invalid cursor: %w", err)
	}

	if c.Offset < 0 {
		return listTagsCursor{}, fmt.Errorf("invalid cursor: negative offset")
	}

	if !isValidSortOrder(c.Sort) {
		return listTagsCursor{}, fmt.Errorf("invalid cursor: unrecognized sort order %q", c.Sort)
	}

	return c, nil
}

// sortTags returns a sorted copy of the tags slice according to the given order.
func sortTags(tags []string, order string) []string {
	sorted := make([]string, len(tags))
	copy(sorted, tags)

	switch order {
	case SortAlphabetical:
		sort.Strings(sorted)
	case SortAlphabeticalDesc:
		sort.Sort(sort.Reverse(sort.StringSlice(sorted)))
	case SortSemver:
		sortSemver(sorted, false)
	case SortSemverDesc:
		sortSemver(sorted, true)
	}

	return sorted
}

// ensureVPrefix adds a "v" prefix if not already present, for semver comparison.
func ensureVPrefix(tag string) string {
	if !strings.HasPrefix(tag, "v") {
		return "v" + tag
	}
	return tag
}

// sortSemver sorts tags with valid semver first, non-semver appended alphabetically.
func sortSemver(tags []string, descending bool) {
	var semverTags, nonSemverTags []string

	for _, tag := range tags {
		if semver.IsValid(ensureVPrefix(tag)) {
			semverTags = append(semverTags, tag)
		} else {
			nonSemverTags = append(nonSemverTags, tag)
		}
	}

	sort.Slice(semverTags, func(i, j int) bool {
		cmp := semver.Compare(ensureVPrefix(semverTags[i]), ensureVPrefix(semverTags[j]))
		if descending {
			return cmp > 0
		}
		return cmp < 0
	})

	if descending {
		sort.Sort(sort.Reverse(sort.StringSlice(nonSemverTags)))
	} else {
		sort.Strings(nonSemverTags)
	}

	copy(tags, semverTags)
	copy(tags[len(semverTags):], nonSemverTags)
}

// paginateTags returns a page of tags from the given offset with the given limit.
// nextOffset is 0 when there are no more pages.
func paginateTags(tags []string, offset, limit int) (page []string, nextOffset int) {
	if offset >= len(tags) {
		return nil, 0
	}

	end := offset + limit
	if end >= len(tags) {
		return tags[offset:], 0
	}

	return tags[offset:end], end
}
