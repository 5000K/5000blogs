package service

// NewPost constructs a Post with explicit metadata and rendered content.
// Intended for use in tests and tooling; not used by the normal ingestion flow.
func NewPost(path string, metadata *Metadata, content []byte) *Post {
	return &Post{
		path:     path,
		metadata: metadata,
		contents: &content,
	}
}
