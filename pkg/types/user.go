package types

// User represents an authenticated, human user in mcpjungle
// A user has lesser privileges than an Admin.
// They can consume mcpjungle but not necessarily manage it.
type User struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
}
