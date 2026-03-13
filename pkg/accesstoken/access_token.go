package accesstoken

import (
	"fmt"
	"os"
	"strings"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// Input defines how to resolve an effective access token.
type Input struct {
	Inline string
	Ref    types.AccessTokenRef
}

// Resolve returns the effective access token using the following precedence:
// 1. Inline token if it is not empty
// 2. Environment variable specified in Ref.Env
// 3. File specified in Ref.File
// If none are present, it returns an empty token and no error.
func Resolve(input Input) (string, error) {
	if input.Inline != "" {
		return input.Inline, nil
	}

	if input.Ref.Env != "" {
		value, ok := os.LookupEnv(input.Ref.Env)
		if ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed, nil
			}
		}
		if input.Ref.File == "" {
			return "", fmt.Errorf("environment variable %s is not set or empty", input.Ref.Env)
		}
	}

	if input.Ref.File != "" {
		data, err := os.ReadFile(input.Ref.File)
		if err != nil {
			return "", fmt.Errorf("failed to read access token file %s: %w", input.Ref.File, err)
		}
		trimmed := strings.TrimSpace(string(data))
		if trimmed == "" {
			return "", fmt.Errorf("access token file %s is empty", input.Ref.File)
		}
		return trimmed, nil
	}

	return "", nil
}
