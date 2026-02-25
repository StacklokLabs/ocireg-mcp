package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		offset int
		sort   string
	}{
		{"zero offset alphabetical", 0, SortAlphabetical},
		{"positive offset semver", 100, SortSemver},
		{"large offset desc", 9999, SortAlphabeticalDesc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := encodeCursor(tt.offset, tt.sort)
			decoded, err := decodeCursor(cursor)
			require.NoError(t, err)
			assert.Equal(t, tt.offset, decoded.Offset)
			assert.Equal(t, tt.sort, decoded.Sort)
		})
	}
}

func TestDecodeCursor_Errors(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"bad base64", "!!!not-base64!!!"},
		{"bad json", "bm90LWpzb24"},
		{"negative offset", encodeCursor(-1, SortAlphabetical)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeCursor(tt.cursor)
			assert.Error(t, err)
		})
	}
}

func TestIsValidSortOrder(t *testing.T) {
	assert.True(t, isValidSortOrder(SortAlphabetical))
	assert.True(t, isValidSortOrder(SortAlphabeticalDesc))
	assert.True(t, isValidSortOrder(SortSemver))
	assert.True(t, isValidSortOrder(SortSemverDesc))
	assert.False(t, isValidSortOrder("invalid"))
	assert.False(t, isValidSortOrder(""))
}

func TestSortTags_Alphabetical(t *testing.T) {
	tags := []string{"zebra", "apple", "mango"}
	result := sortTags(tags, SortAlphabetical)
	assert.Equal(t, []string{"apple", "mango", "zebra"}, result)
	// Original unchanged
	assert.Equal(t, []string{"zebra", "apple", "mango"}, tags)
}

func TestSortTags_AlphabeticalDesc(t *testing.T) {
	tags := []string{"apple", "mango", "zebra"}
	result := sortTags(tags, SortAlphabeticalDesc)
	assert.Equal(t, []string{"zebra", "mango", "apple"}, result)
}

func TestSortTags_Semver(t *testing.T) {
	tags := []string{"v2.0.0", "v1.0.0", "latest"}
	result := sortTags(tags, SortSemver)
	assert.Equal(t, []string{"v1.0.0", "v2.0.0", "latest"}, result)
}

func TestSortTags_SemverDesc(t *testing.T) {
	tags := []string{"v1.0.0", "v2.0.0", "latest", "beta"}
	result := sortTags(tags, SortSemverDesc)
	// Semver tags descending, then non-semver tags descending alphabetically
	assert.Equal(t, []string{"v2.0.0", "v1.0.0", "latest", "beta"}, result)
}

func TestSortTags_SemverMixedVPrefix(t *testing.T) {
	tags := []string{"2.0.0", "v1.0.0", "1.5.0"}
	result := sortTags(tags, SortSemver)
	assert.Equal(t, []string{"v1.0.0", "1.5.0", "2.0.0"}, result)
}

func TestSortTags_AllNonSemver(t *testing.T) {
	tags := []string{"latest", "nightly", "beta"}
	result := sortTags(tags, SortSemver)
	assert.Equal(t, []string{"beta", "latest", "nightly"}, result)
}

func TestSortTags_Empty(t *testing.T) {
	result := sortTags(nil, SortAlphabetical)
	assert.Empty(t, result)
}

func TestSortTags_Single(t *testing.T) {
	result := sortTags([]string{"v1.0.0"}, SortSemver)
	assert.Equal(t, []string{"v1.0.0"}, result)
}

func TestPaginateTags_FirstPage(t *testing.T) {
	tags := []string{"a", "b", "c", "d", "e"}
	page, next := paginateTags(tags, 0, 2)
	assert.Equal(t, []string{"a", "b"}, page)
	assert.Equal(t, 2, next)
}

func TestPaginateTags_MiddlePage(t *testing.T) {
	tags := []string{"a", "b", "c", "d", "e"}
	page, next := paginateTags(tags, 2, 2)
	assert.Equal(t, []string{"c", "d"}, page)
	assert.Equal(t, 4, next)
}

func TestPaginateTags_LastPage(t *testing.T) {
	tags := []string{"a", "b", "c", "d", "e"}
	page, next := paginateTags(tags, 4, 2)
	assert.Equal(t, []string{"e"}, page)
	assert.Equal(t, 0, next)
}

func TestPaginateTags_ExactBoundary(t *testing.T) {
	tags := []string{"a", "b", "c", "d"}
	page, next := paginateTags(tags, 0, 4)
	assert.Equal(t, []string{"a", "b", "c", "d"}, page)
	assert.Equal(t, 0, next)
}

func TestPaginateTags_LimitExceedsRemaining(t *testing.T) {
	tags := []string{"a", "b", "c"}
	page, next := paginateTags(tags, 1, 100)
	assert.Equal(t, []string{"b", "c"}, page)
	assert.Equal(t, 0, next)
}

func TestPaginateTags_OffsetBeyondEnd(t *testing.T) {
	tags := []string{"a", "b"}
	page, next := paginateTags(tags, 10, 2)
	assert.Nil(t, page)
	assert.Equal(t, 0, next)
}
