package service

// NewPost constructs a Post with explicit metadata and rendered content.
// Intended for use in tests and tooling; not used by the normal ingestion flow.
// The slug is derived from path using slugFromPath (basename without extension).
func NewPost(path string, metadata *Metadata, content []byte) *Post {
	p := &Post{
		path:     path,
		slug:     slugFromPath(path),
		metadata: metadata,
	}
	if content != nil {
		p.contents = &content
	}
	return p
}
