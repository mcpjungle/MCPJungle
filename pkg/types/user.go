package types

// UserRole represents the role of a user in the MCPJungle system.
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

// UserConfig describes the JSON configuration for creating a user.
type UserConfig struct {
	// Username is the unique name of the user to create (mandatory)
	Username string `json:"name"`

	// AccessToken allows you to provide a custom access token the user can use
	// to authenticate with MCPJungle.
	// It is not recommended to use this field in production environments, since
	// a hard-coded access token can be a security risk.
	// Instead, use the AccessTokenRef field to load the access token from
	// a secure location.
	AccessToken string `json:"access_token,omitempty"`

	// AccessTokenRef allows you to specify how to load the access token from
	// an external source. Use this in production scenarios, especially when
	// you want to commit the user configuration to version control.
	AccessTokenRef AccessTokenRef `json:"access_token_ref"`
}

// User represents an authenticated, human user in mcpjungle
// A user has lesser privileges than an Admin.
// They can consume mcpjungle but not necessarily manage it.
type User struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type CreateOrUpdateUserRequest struct {
	Username    string `json:"username"`
	AccessToken string `json:"access_token,omitempty"`
}

type CreateOrUpdateUserResponse struct {
	Username    string `json:"username"`
	Role        string `json:"role"`
	AccessToken string `json:"access_token"`
}
