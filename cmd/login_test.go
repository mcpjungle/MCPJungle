package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestLoginCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "login [access_token]", loginCmd.Use)
		testhelpers.AssertEqual(t, "Log in to MCPJungle (Production mode)", loginCmd.Short)
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "6"},
		}
		testhelpers.TestCommandAnnotations(t, loginCmd.Annotations, annotationTests)
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, loginCmd.Args)
	})
}

func TestUserRoleConstants(t *testing.T) {
	t.Run("user_role_constants", func(t *testing.T) {
		testhelpers.AssertEqual(t, "admin", string(types.UserRoleAdmin))
		testhelpers.AssertEqual(t, "user", string(types.UserRoleUser))
	})
}
