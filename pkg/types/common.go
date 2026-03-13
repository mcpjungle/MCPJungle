package types

// AccessTokenRef describes how to load a secret access token from an external source.
type AccessTokenRef struct {
	// Env indicates that the access token should be loaded from an environment variable.
	// The value is the name of the environment variable.
	Env string `json:"env"`

	// File indicates that the access token should be loaded from a file.
	// The value is the path to the file.
	File string `json:"file"`
}
