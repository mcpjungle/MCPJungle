package types

// UserRole represents the role of a user in the MCPJungle system.
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

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

type CreateUserResponse struct {
	Username    string `json:"username"`
	Role        string `json:"role"`
	AccessToken string `json:"access_token"`
}
