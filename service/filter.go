package service

// PostFilter represents the filtering criteria for retrieving posts.
type PostFilter struct {
	Tags  []string
	Query string
}
