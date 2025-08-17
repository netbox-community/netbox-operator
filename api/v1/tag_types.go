package v1

type Tag struct {
	// +optional
	// Name of the tag
	Name string `json:"name,omitempty"`

	// +optional
	// Slug of the tag
	Slug string `json:"slug,omitempty"`
}
