package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
)

func (s *Server) createUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Username    string   `json:"username"`
			AccessToken string   `json:"access_token"`
			AllowList   []string `json:"allow_list"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		newUser, err := s.userService.CreateUser(&model.User{
			Username:    input.Username,
			AccessToken: input.AccessToken,
			AllowList:   datatypes.JSON(mustMarshalStringSlice(input.AllowList)),
		})
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := &types.CreateOrUpdateUserResponse{
			Username:    newUser.Username,
			Role:        string(newUser.Role),
			AccessToken: newUser.AccessToken,
			AllowList:   parseAllowList(newUser.AllowList),
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func (s *Server) listUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := s.userService.ListUsers()
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := make([]*types.User, len(users))
		for i, u := range users {
			resp[i] = &types.User{
				Username:  u.Username,
				Role:      string(u.Role),
				AllowList: parseAllowList(u.AllowList),
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) updateUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
			return
		}

		var input struct {
			AccessToken       string   `json:"access_token"`
			RotateAccessToken bool     `json:"rotate_access_token"`
			AllowList         []string `json:"allow_list"`
			UpdateAllowList   bool     `json:"update_allow_list"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updatedUser, err := s.userService.UpdateUser(user.UpdateUserInput{
			Username:          username,
			AccessToken:       input.AccessToken,
			RotateAccessToken: input.RotateAccessToken,
			AllowList:         input.AllowList,
			UpdateAllowList:   input.UpdateAllowList,
		})
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := &types.CreateOrUpdateUserResponse{
			Username:    updatedUser.Username,
			Role:        string(updatedUser.Role),
			AccessToken: updatedUser.AccessToken,
			AllowList:   parseAllowList(updatedUser.AllowList),
		}
		c.JSON(http.StatusOK, resp)
	}
}

// parseAllowList decodes a JSON-stored allow list into a string slice.
// null / missing DB value → ["*"]  (wildcard default for unset users)
// []                      → []     (explicitly no access — never nil, so JSON encodes as [])
// ["a","b"]               → ["a","b"]
func parseAllowList(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return []string{"*"}
	}
	// Use a pre-initialized slice so json.Marshal always emits [] not null.
	list := make([]string, 0)
	if err := json.Unmarshal(raw, &list); err != nil {
		return []string{"*"}
	}
	return list
}

// mustMarshalStringSlice marshals a string slice to JSON bytes; falls back to wildcard on error.
func mustMarshalStringSlice(s []string) []byte {
	if s == nil {
		return []byte(`["*"]`)
	}
	b, err := json.Marshal(s)
	if err != nil {
		return []byte(`["*"]`)
	}
	return b
}

func (s *Server) deleteUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
			return
		}

		err := s.userService.DeleteUser(username)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *Server) whoAmIHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		u, ok := currentUser.(*model.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user from context"})
			return
		}

		resp := types.User{
			Username: u.Username,
			Role:     string(u.Role),
		}
		c.JSON(http.StatusOK, resp)
	}
}
