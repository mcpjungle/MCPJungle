package group_test

import (
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/mcpjungle/mcpjungle/internal/model"
	groupsvc "github.com/mcpjungle/mcpjungle/internal/service/group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.User{}))
	return db
}

func mustJSON(v []string) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestResolveAllowList_ExplicitUserOverridesGroup(t *testing.T) {
	db := newTestDB(t)
	svc := groupsvc.NewGroupService(db)

	g := model.Group{Name: "eng", AllowList: mustJSON([]string{"atlassian"})}
	db.Create(&g)

	u := model.User{Username: "thi", Role: "user", AccessToken: "tok1",
		AllowList: mustJSON([]string{"github"}), GroupID: &g.ID}
	db.Create(&u)

	result := svc.ResolveAllowList(&u)
	assert.Equal(t, []string{"github"}, result)
}

func TestResolveAllowList_InheritsGroup(t *testing.T) {
	db := newTestDB(t)
	svc := groupsvc.NewGroupService(db)

	g := model.Group{Name: "eng", AllowList: mustJSON([]string{"atlassian", "github"})}
	db.Create(&g)

	u := model.User{Username: "thi", Role: "user", AccessToken: "tok2", GroupID: &g.ID}
	db.Create(&u)

	result := svc.ResolveAllowList(&u)
	assert.Equal(t, []string{"atlassian", "github"}, result)
}

func TestResolveAllowList_DefaultWildcard(t *testing.T) {
	db := newTestDB(t)
	svc := groupsvc.NewGroupService(db)

	u := model.User{Username: "solo", Role: "user", AccessToken: "tok3"}
	db.Create(&u)

	result := svc.ResolveAllowList(&u)
	assert.Equal(t, []string{"*"}, result)
}

func TestDeleteGroup_WithMembersReturnsError(t *testing.T) {
	db := newTestDB(t)
	svc := groupsvc.NewGroupService(db)

	g := model.Group{Name: "eng", AllowList: mustJSON([]string{"*"})}
	db.Create(&g)
	u := model.User{Username: "member", Role: "user", AccessToken: "tok4", GroupID: &g.ID}
	db.Create(&u)

	err := svc.DeleteGroup("eng")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 member")
}
