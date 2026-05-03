package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	groupsvc "github.com/mcpjungle/mcpjungle/internal/service/group"
)

type groupResponse struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	AllowList   []string `json:"allow_list"`
	MemberCount int      `json:"member_count"`
}

func groupToResponse(g model.Group, count int) groupResponse {
	al := make([]string, 0)
	if len(g.AllowList) > 0 {
		_ = json.Unmarshal(g.AllowList, &al)
	}
	if al == nil {
		al = []string{"*"}
	}
	return groupResponse{
		Name:        g.Name,
		Description: g.Description,
		AllowList:   al,
		MemberCount: count,
	}
}

func (s *Server) listGroupsHandler(c *gin.Context) {
	groups, err := s.groupService.ListGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]groupResponse, 0, len(groups))
	for _, g := range groups {
		resp = append(resp, groupToResponse(g, s.groupService.MemberCount(g.ID)))
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) createGroupHandler(c *gin.Context) {
	var body struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		AllowList   []string `json:"allow_list"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	encoded, _ := json.Marshal(body.AllowList)
	g, err := s.groupService.CreateGroup(model.Group{
		Name:        body.Name,
		Description: body.Description,
		AllowList:   encoded,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, groupToResponse(*g, 0))
}

func (s *Server) updateGroupHandler(c *gin.Context) {
	name := c.Param("name")
	var body struct {
		Description     string   `json:"description"`
		AllowList       []string `json:"allow_list"`
		UpdateAllowList bool     `json:"update_allow_list"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := s.groupService.UpdateGroup(name, groupsvc.UpdateGroupInput{
		Description:     body.Description,
		AllowList:       body.AllowList,
		UpdateAllowList: body.UpdateAllowList,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Propagate new allow_list to self-service tokens of group members who have no
	// explicit per-user override (allow_list IS NULL in DB).
	if body.UpdateAllowList {
		newAllowList, _ := json.Marshal(body.AllowList)
		if members, merr := s.groupService.GetGroupMembers(g.ID); merr == nil {
			for _, u := range members {
				// User inherits from group if allow_list is NULL or empty array [].
				// Matches ResolveAllowList logic: empty decoded slice = fall through to group.
				var decoded []string
				inheriting := len(u.AllowList) == 0 ||
					(json.Unmarshal(u.AllowList, &decoded) == nil && len(decoded) == 0)
				if inheriting {
					_ = s.mcpClientService.UpdateAllowListByOwner(u.Username, newAllowList)
				}
			}
		}
	}

	c.JSON(http.StatusOK, groupToResponse(*g, s.groupService.MemberCount(g.ID)))
}

func (s *Server) deleteGroupHandler(c *gin.Context) {
	name := c.Param("name")
	if err := s.groupService.DeleteGroup(name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) assignUserToGroupHandler(c *gin.Context) {
	groupName := c.Param("name")
	var body struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
		return
	}
	if err := s.groupService.AssignUser(body.Username, groupName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) removeUserFromGroupHandler(c *gin.Context) {
	username := c.Param("username")
	if err := s.groupService.RemoveUserFromGroup(username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
