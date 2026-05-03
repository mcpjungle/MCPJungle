# Groups Feature — Option 2 Implementation Checklist

**Approach**: Groups as a UI-managed admin construct backed by a thin Go service layer.
The MCP proxy, CLI commands, and all existing `/api/v0/*` endpoints remain untouched.
Group membership resolves a user's effective allow-list at client-token creation time.

---

## Architecture Overview

```
Admin sets Group.allow_list
        │
        ▼
User assigned to Group  ←─── or ─── User has explicit allow_list (overrides group)
        │
        ▼
createSelfClientHandler calls ResolveAllowList(user)
        │
        ▼
McpClient created with resolved allow_list
        │
        ▼
MCP proxy enforces allow_list at request time (unchanged)
```

**Allow-list resolution priority:**
1. `User.allow_list` (explicit, set by admin) — highest priority
2. `Group.allow_list` (inherited when user has no explicit list)
3. Default `["*"]` — wildcard if neither is set

---

## Phase 1 — Backend: Data Model

### 1.1 New file: `internal/model/group.go`
- [x] Define `Group` struct with GORM model
  ```go
  type Group struct {
      gorm.Model
      Name        string         `json:"name" gorm:"uniqueIndex;not null"`
      Description string         `json:"description"`
      AllowList   datatypes.JSON `json:"allow_list" gorm:"type:jsonb"`
  }
  ```
- [x] Confirm `datatypes.JSON` import (`gorm.io/datatypes`)

### 1.2 Update `internal/model/user.go`
- [x] Add nullable `GroupID *uint` field with GORM index
  ```go
  GroupID *uint  `json:"group_id" gorm:"index"`
  Group   *Group `json:"group,omitempty" gorm:"foreignKey:GroupID"`
  ```
- [x] Verify existing users get `group_id = NULL` with no migration errors (GORM auto-migrate adds nullable column safely)

### 1.3 Database migration
- [x] Run the app once after model changes — GORM auto-migrate creates `groups` table and `group_id` column on `users`
- [x] Verify with `sqlite3 mcpjungle.db ".schema groups"` and `PRAGMA table_info(users)`
- [x] Confirm existing `admin`, `thi`, `thi2` users are unaffected (group_id = NULL, allow_list unchanged)

---

## Phase 2 — Backend: Service Layer

### 2.1 New file: `internal/service/group/group.go`

- [x] Define `GroupService` struct with `*gorm.DB`
- [x] `NewGroupService(db *gorm.DB) *GroupService`
- [x] `CreateGroup(input model.Group) (*model.Group, error)`
  - Validate `Name` not empty
  - Default `AllowList` to `["*"]` if not provided
- [x] `UpdateGroup(name string, input UpdateGroupInput) (*model.Group, error)`
  ```go
  type UpdateGroupInput struct {
      Description     string
      AllowList       []string
      UpdateAllowList bool
  }
  ```
  - Same `UpdateAllowList` bool pattern as `UpdateUserInput` to distinguish empty vs unset
- [x] `DeleteGroup(name string) error`
  - Reject if group has members (return descriptive error)
  - Or: unset `GroupID` on all members before deleting (choose one, document the choice)
- [x] `ListGroups() ([]model.Group, error)` — preload member count
- [x] `GetGroup(name string) (*model.Group, error)`
- [x] `AssignUser(username string, groupName string) error`
  - Set `User.GroupID` to the group's ID
  - Clear `User.AllowList` only if admin explicitly requests (keep explicit allow_list by default)
- [x] `RemoveUserFromGroup(username string) error`
  - Set `User.GroupID = NULL`
- [x] `ResolveAllowList(user *model.User) []string` — **core logic**
  ```go
  // If user has explicit allow_list (non-null in DB), return it.
  // Else if user belongs to a group, return group's allow_list.
  // Else return ["*"] wildcard.
  func (s *GroupService) ResolveAllowList(user *model.User) []string
  ```
- [x] Unit tests in `internal/service/group/group_test.go`
  - [x] User with explicit allow_list ignores group
  - [x] User with no allow_list inherits group allow_list
  - [x] User with no allow_list and no group returns `["*"]`
  - [x] DeleteGroup with members returns error (or clears members — whichever is chosen)

---

## Phase 3 — Backend: API Handlers

### 3.1 New file: `internal/api/groups.go`

- [x] `listGroupsHandler` — `GET /api/v0/groups`
  - Returns `[]GroupResponse` with member count
  ```go
  type GroupResponse struct {
      Name        string   `json:"name"`
      Description string   `json:"description"`
      AllowList   []string `json:"allow_list"`
      MemberCount int      `json:"member_count"`
  }
  ```
- [x] `createGroupHandler` — `POST /api/v0/groups`
  ```json
  { "name": "eng", "description": "Engineering team", "allow_list": ["github", "atlassian"] }
  ```
- [x] `updateGroupHandler` — `PUT /api/v0/groups/:name`
- [x] `deleteGroupHandler` — `DELETE /api/v0/groups/:name`
- [x] `assignUserHandler` — `POST /api/v0/groups/:name/members`
  ```json
  { "username": "thi" }
  ```
- [x] `removeUserHandler` — `DELETE /api/v0/groups/:name/members/:username`
- [x] All handlers: admin-only middleware, consistent error responses (same pattern as `users.go`)
- [x] Handler tests in `internal/api/groups_test.go`

### 3.2 Update `internal/api/server.go`

- [x] Register group routes under the admin middleware group:
  ```go
  // Group management (admin only, enterprise mode)
  adminGroup.POST("/groups", s.createGroupHandler)
  adminGroup.GET("/groups", s.listGroupsHandler)
  adminGroup.PUT("/groups/:name", s.updateGroupHandler)
  adminGroup.DELETE("/groups/:name", s.deleteGroupHandler)
  adminGroup.POST("/groups/:name/members", s.assignUserHandler)
  adminGroup.DELETE("/groups/:name/members/:username", s.removeUserHandler)
  ```
- [x] Wire `groupService` into `Server` struct and constructor

### 3.3 Update `internal/api/mcp_clients.go`

- [x] Inject `groupService` into `createSelfClientHandler`
- [x] Replace direct `userModel.AllowList` read with `groupService.ResolveAllowList(userModel)`
- [x] Verify the null-check for unset user AllowList is now inside `ResolveAllowList` (remove duplication)

### 3.4 Update `internal/api/users.go`

- [x] Include `group_id` and `group_name` in `listUsersHandler` response
  ```go
  type UserResponse struct {
      Username  string   `json:"username"`
      Role      string   `json:"role"`
      AllowList []string `json:"allow_list"`
      GroupID   *uint    `json:"group_id,omitempty"`
      GroupName string   `json:"group_name,omitempty"`
  }
  ```
- [x] Preload `Group` when listing users so `Group.Name` is available

### 3.5 Update `pkg/types/user.go`

- [x] Add `GroupID *uint` and `GroupName string` to `User` type (used by CLI client if any)

---

## Phase 4 — Frontend: Types and API Client

### 4.1 Update `ui/src/lib/types.ts`

- [x] Add `Group` type:
  ```typescript
  export type Group = {
    name: string
    description: string
    allow_list: string[]
    member_count: number
  }
  ```
- [x] Update `CurrentUser` to include optional group info:
  ```typescript
  group_id?: number
  group_name?: string
  ```

### 4.2 Update `ui/src/lib/api.ts`

- [x] `api.groups(token?)` → `GET /api/v0/groups` → `Group[]`
- [x] `api.createGroup(body, token?)` → `POST /api/v0/groups` → `Group`
- [x] `api.updateGroup(name, body, token?)` → `PUT /api/v0/groups/:name` → `Group`
- [x] `api.deleteGroup(name, token?)` → `DELETE /api/v0/groups/:name`
- [x] `api.assignUserToGroup(groupName, username, token?)` → `POST /api/v0/groups/:name/members`
- [x] `api.removeUserFromGroup(groupName, username, token?)` → `DELETE /api/v0/groups/:name/members/:username`

---

## Phase 5 — Frontend: Groups Page

### 5.1 New file: `ui/src/pages/GroupsPage.tsx`

#### Table view
- [x] Columns: `NAME`, `DESCRIPTION`, `SERVER ACCESS`, `MEMBERS`, `ACTIONS`
- [x] `SERVER ACCESS` cell: use existing `AllowListBadges` component
- [x] `MEMBERS` cell: number badge (count)
- [x] `ACTIONS`: Edit button, Delete button (with confirm dialog)
- [x] "New group" button top-right (same pattern as UsersPage)
- [x] Empty state when no groups exist

#### Create/Edit drawer (inline panel, same UX as UsersPage permissions drawer)
- [x] Fields: Name (create only, readonly on edit), Description
- [x] `AllowListEditor` for server allow-list (fetch servers list same as ClientsPage)
- [x] Save / Cancel buttons

#### Members sub-section inside Edit drawer
- [x] List current members: username + role chips
- [x] "Add member" — searchable select of existing users not already in this group
- [x] Remove member button per row (with inline confirmation)
- [x] Show warning if a member has an explicit allow-list that overrides the group's list

#### State management
- [x] TanStack Query for `useQuery(['groups'], api.groups)`
- [x] Mutations: `createGroup`, `updateGroup`, `deleteGroup`, `assignUser`, `removeUser` — all invalidate `['groups']` and `['users']` on success
- [x] Drawer state: `mode: null | "create" | "edit"`, `drawerTarget: Group | null`

### 5.2 Update `ui/src/pages/UsersPage.tsx`

- [x] Add `GROUP` column to users table
  - Show group name badge if assigned, `—` if no group
- [x] In Permissions drawer: show inherited/override hint
  ```
  If user has no explicit allow_list AND belongs to a group:
    "Inheriting from group: eng — [atlassian] [github]"
    Button: "Override for this user"
  If user has explicit allow_list:
    Show AllowListEditor (current behavior)
    Button: "Reset to group default" (sets allow_list = null, clears explicit override)
  ```
- [x] "Reset to group default" sets `User.AllowList = null` via `updateUser` with `update_allow_list: true, allow_list: null`
  - Backend: detect `null` in request body = unset the explicit override (set DB field to NULL)
  - Requires a backend update: distinguish `null` (clear explicit list) from `[]` (no access)

### 5.3 Add route — `ui/src/lib/router.tsx`

- [x] Add route:
  ```typescript
  { path: "groups", element: <GroupsPage /> }
  ```

### 5.4 Add nav link — `ui/src/components/layout/NavSidebar.tsx`

- [x] Add "Groups" nav item between Users and Settings
- [x] Icon: use same icon library pattern as other nav items (e.g. `Users` or `Layers` from lucide-react)
- [x] Show only in enterprise mode (same guard as Users nav item)

---

## Phase 6 — Backend: Allow-list "Reset to Group" Support

### 6.1 Update `internal/service/user/user.go`

- [x] Add `ClearAllowList bool` to `UpdateUserInput`
  ```go
  type UpdateUserInput struct {
      Username          string
      AccessToken       string
      RotateAccessToken bool
      AllowList         []string
      UpdateAllowList   bool
      ClearAllowList    bool // if true, set AllowList = NULL (revert to group inheritance)
  }
  ```
- [x] In `UpdateUser`: if `ClearAllowList` is true, set `user.AllowList = nil` (NULL in DB)

### 6.2 Update `internal/api/users.go`

- [x] In `updateUserHandler`: detect `"allow_list": null` in JSON body → set `ClearAllowList = true`
  - Use `json.RawMessage` or a pointer `*[]string` to distinguish `null` from absent key

### 6.3 Update `ui/src/lib/api.ts`

- [x] `updateUser` body: allow `allow_list: null` to send explicit null
  ```typescript
  type UpdateUserInput = {
    allow_list?: string[] | null   // null = clear override, [] = no access, [...] = explicit list
    update_allow_list?: boolean
    ...
  }
  ```

---

## Phase 7 — Integration and Testing

- [x] `go build ./...` — compiles cleanly
- [x] `go test ./internal/service/group/...` — unit tests pass
- [x] `go test ./internal/api/...` — handler tests pass (include groups)
- [x] `go test ./...` — full suite green
- [x] `golangci-lint run` — no new lint errors
- [x] Manual flow test:
  - [x] Create group "eng" with allow_list `["atlassian", "github"]`
  - [x] Assign user `thi` to group "eng" (thi has no explicit allow_list)
  - [x] Verify `thi` settings page shows inherited servers `[atlassian] [github]`
  - [x] Create self-service MCP client as `thi` → verify client allow_list = `["atlassian", "github"]`
  - [x] Set explicit override for `thi`: allow_list = `["atlassian"]`
  - [x] Verify client creation now uses `["atlassian"]` (ignores group)
  - [x] Reset `thi` to group default → allow_list = NULL → next client creation uses group list again
  - [x] Delete group "eng" with members → verify error or auto-unassign (whichever is implemented)

---

## Phase 8 — Build and Release

- [x] `cd ui && npm run build` — no TypeScript errors, build succeeds
- [x] `go build ./...` — no compile errors
- [x] Update `DESIGN.md` — add Groups section describing the data model and UX
- [x] Commit all changes on a new branch `feat/groups-ui`
- [x] Open PR against `main` on `dinhdobathi1992/MCPJungle`
- [x] After merge, tag `v0.2.0` and create GitHub release

---

## Files Touched Summary

| File | Change |
|---|---|
| `internal/model/group.go` | NEW — Group struct |
| `internal/model/user.go` | ADD GroupID, Group fields |
| `internal/service/group/group.go` | NEW — GroupService |
| `internal/service/group/group_test.go` | NEW — unit tests |
| `internal/service/user/user.go` | ADD ClearAllowList support |
| `internal/api/groups.go` | NEW — CRUD + member handlers |
| `internal/api/groups_test.go` | NEW — handler tests |
| `internal/api/server.go` | ADD group routes, wire groupService |
| `internal/api/mcp_clients.go` | USE ResolveAllowList instead of direct field |
| `internal/api/users.go` | ADD group fields to response, handle null allow_list |
| `pkg/types/user.go` | ADD GroupID, GroupName fields |
| `ui/src/lib/types.ts` | ADD Group type, update CurrentUser |
| `ui/src/lib/api.ts` | ADD group API functions |
| `ui/src/pages/GroupsPage.tsx` | NEW — Groups management page |
| `ui/src/pages/UsersPage.tsx` | ADD group column, inheritance hint, reset button |
| `ui/src/lib/router.tsx` | ADD /ui/groups route |
| `ui/src/components/layout/NavSidebar.tsx` | ADD Groups nav link |

---

## Open Decisions (resolve before starting)

1. **Delete group with members**: error and require manual unassign first, or silently unassign all members?
   - Recommendation: error + list affected usernames — safer, avoids surprise access changes

2. **Explicit override vs group list in UI**: when admin sets an explicit allow_list for a user who is in a group, should the UI show a warning that the user's explicit list overrides the group?
   - Recommendation: yes — show an amber banner in the Permissions drawer

3. **Multiple groups per user**: current design allows only one group per user (single `group_id`). If multi-group membership is needed later, the `group_id` foreign key must become a join table.
   - Recommendation: ship single-group first; multi-group is a future migration

4. **Group allow_list scope**: does a group's allow_list restrict access to server *names* only (current model) or also tool-level granularity?
   - Recommendation: server names only for v0.2.0, consistent with existing allow_list semantics
