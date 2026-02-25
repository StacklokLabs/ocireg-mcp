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
