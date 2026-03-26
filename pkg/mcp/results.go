package mcp

// ListTagsResult is the structured result for the list_tags tool.
type ListTagsResult struct {
	Tags       []string `json:"tags"`
	TotalCount int      `json:"totalCount"`
	NextCursor string   `json:"nextCursor,omitempty"`
	Sort       string   `json:"sort"`
}

// ImageInfoResult is the structured result for the get_image_info tool.
type ImageInfoResult struct {
	Digest       string `json:"digest"`
	Size         int64  `json:"size"`
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Created      string `json:"created"`
	Layers       int    `json:"layers"`
}

// ReferrerDescriptor describes a single referrer artifact attached to an image.
type ReferrerDescriptor struct {
	MediaType    string            `json:"mediaType"`
	Digest       string            `json:"digest"`
	Size         int64             `json:"size"`
	ArtifactType string            `json:"artifactType,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// ListReferrersResult is the structured result for the list_referrers tool.
type ListReferrersResult struct {
	Referrers []ReferrerDescriptor `json:"referrers"`
	Count     int                  `json:"count"`
}

// ReferrerContentMetadata is the structured metadata for the get_referrer_content tool.
type ReferrerContentMetadata struct {
	ContentType     string `json:"contentType,omitempty"`
	Format          string `json:"format,omitempty"`
	PredicateType   string `json:"predicateType,omitempty"`
	DecodedFromDSSE bool   `json:"decodedFromDsse"`
	Size            int    `json:"size"`
	Truncated       bool   `json:"truncated"`
}
