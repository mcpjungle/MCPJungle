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
			Username    string    `json:"username"`
			AccessToken string    `json:"access_token"`
			AllowList   *[]string `json:"allow_list"` // pointer: nil = not provided (inherit from group)
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var allowList datatypes.JSON
		if input.AllowList != nil {
			b, _ := json.Marshal(*input.AllowList)
			allowList = datatypes.JSON(b)
		}

		newUser, err := s.userService.CreateUser(&model.User{
			Username:    input.Username,
			AccessToken: input.AccessToken,
			AllowList:   allowList,
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
		users, err := s.userService.ListUsersWithGroup()
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := make([]*types.User, len(users))
		for i, u := range users {
			groupName := ""
			if u.Group != nil {
				groupName = u.Group.Name
			}
			resp[i] = &types.User{
				Username:  u.Username,
				Role:      string(u.Role),
				AllowList: parseAllowList(u.AllowList),
				GroupID:   u.GroupID,
				GroupName: groupName,
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

		var body struct {
			AccessToken       string          `json:"access_token"`
			RotateAccessToken bool            `json:"rotate_access_token"`
			AllowList         json.RawMessage `json:"allow_list"` // RawMessage to detect null vs absent
			UpdateAllowList   bool            `json:"update_allow_list"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		input := user.UpdateUserInput{
			Username:          username,
			AccessToken:       body.AccessToken,
			RotateAccessToken: body.RotateAccessToken,
		}

		if body.UpdateAllowList {
			if string(body.AllowList) == "null" || len(body.AllowList) == 0 {
				input.ClearAllowList = true
			} else {
				var list []string
				if err := json.Unmarshal(body.AllowList, &list); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid allow_list"})
					return
				}
				input.AllowList = list
				input.UpdateAllowList = true
			}
		}

		updatedUser, err := s.userService.UpdateUser(input)
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
// null / missing DB value → nil      (no explicit override; omitted from JSON via omitempty)
// []                      → []       (explicitly no access)
// ["a","b"]               → ["a","b"]
func parseAllowList(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return nil
	}
	list := make([]string, 0)
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}
	return list
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
