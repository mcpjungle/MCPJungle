package model

import (
	"fmt"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
)

func TestMcpClient_CheckHasServerAccess(t *testing.T) {
	cases := []struct {
		name   string
		allow  datatypes.JSON
		server string
		want   bool
	}{
		{
			name:   "nil allow list",
			allow:  datatypes.JSON(nil),
			server: "server-1",
			want:   false,
		},
		{
			name:   "empty array",
			allow:  datatypes.JSON("[]"),
			server: "server-1",
			want:   false,
		},
		{
			name:   "global wildcard grants access",
			allow:  datatypes.JSON(fmt.Sprintf(`["%s"]`, types.AllowAllMcpServers)),
			server: "any-server",
			want:   true,
		},
		{
			name:   "global wildcard mixed with other names",
			allow:  datatypes.JSON(fmt.Sprintf(`["%s","server-a","server-b"]`, types.AllowAllMcpServers)),
			server: "any-server",
			want:   true,
		},
		{
			name:   "exact match allowed",
			allow:  datatypes.JSON(`["server-a","server-b"]`),
			server: "server-b",
			want:   true,
		},
		{
			name:   "exact match not present",
			allow:  datatypes.JSON(`["server-a","server-b"]`),
			server: "server-c",
			want:   false,
		},
		{
			name:   "malformed json returns false",
			allow:  datatypes.JSON("not-json"),
			server: "server-a",
			want:   false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &McpClient{AllowList: tc.allow}
			got := client.CheckHasServerAccess(tc.server)
			if got != tc.want {
				t.Fatalf("case %q: CheckHasServerAccess(%q) = %v, want %v", tc.name, tc.server, got, tc.want)
			}
		})
	}
}
