package configresolver

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

const (
	placeholderStart = "${"
	placeholderEnd   = "}"
)

// SecretProvider resolves a secret identified by an opaque key.
type SecretProvider interface {
	Resolve(key string) (string, error)
}

// Resolver expands environment variables and named secret-provider references.
type Resolver struct {
	environment SecretProvider
	providers   map[string]SecretProvider
}

// NewResolver creates a resolver with the supplied named secret providers.
// Environment variable interpolation is always available through ${VAR}.
func NewResolver(providers map[string]SecretProvider) *Resolver {
	return &Resolver{
		environment: envProvider{},
		providers:   providers,
	}
}

// ResolveConfig resolves all supported placeholders in a configuration object.
func ResolveConfig(target any) error {
	return NewResolver(map[string]SecretProvider{
		"keychain": newKeychainProvider(),
	}).Resolve(target)
}

// ResolveEnvVars is retained for callers that only need the legacy ${VAR}
// behavior. It also resolves named secret-provider references.
func ResolveEnvVars(target any) error {
	return ResolveConfig(target)
}

// Resolve walks a configuration object recursively and expands placeholders in
// every string value.
func (r *Resolver) Resolve(target any) error {
	if target == nil {
		return fmt.Errorf("config target must be a non-nil pointer")
	}

	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("config target must be a non-nil pointer")
	}

	return r.resolveConfigValue(value)
}

func (r *Resolver) resolveConfigValue(value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Pointer:
		// Recurse into pointer targets when present; nil pointers are left untouched.
		if value.IsNil() {
			return nil
		}
		return r.resolveConfigValue(value.Elem())
	case reflect.Interface:
		if value.IsNil() {
			return nil
		}

		// Interface values are not directly writable, so resolve a cloned value and
		// replace the interface payload with the resolved copy.
		resolved, err := r.cloneAndResolveValue(value.Elem())
		if err != nil {
			return err
		}
		if value.CanSet() {
			value.Set(resolved)
		}
		return nil
	case reflect.Struct:
		// Walk nested config structs so placeholder expansion applies consistently
		// across the full config object, not just top-level fields.
		for i := range value.NumField() {
			field := value.Field(i)
			if !field.CanSet() {
				continue
			}
			if err := r.resolveConfigValue(field); err != nil {
				return err
			}
		}
		return nil
	case reflect.String:
		if !value.CanSet() {
			return nil
		}

		// Only string values are expanded. Other scalar types are intentionally
		// left unchanged to keep resolution predictable.
		resolved, err := r.expandPlaceholders(value.String())
		if err != nil {
			return err
		}
		value.SetString(resolved)
		return nil
	case reflect.Slice, reflect.Array:
		for i := range value.Len() {
			if err := r.resolveConfigValue(value.Index(i)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if value.IsNil() {
			return nil
		}

		// Map entries are not addressable through reflection, so resolve each value
		// through a writable clone and write it back under the same key.
		for _, key := range value.MapKeys() {
			resolved, err := r.cloneAndResolveValue(value.MapIndex(key))
			if err != nil {
				return err
			}
			value.SetMapIndex(key, resolved)
		}
		return nil
	default:
		return nil
	}
}

func (r *Resolver) cloneAndResolveValue(value reflect.Value) (reflect.Value, error) {
	if !value.IsValid() {
		return value, nil
	}

	// Map and interface elements are often non-settable; this gives recursive
	// resolution a writable copy to operate on.
	clone := reflect.New(value.Type()).Elem()
	clone.Set(value)

	if err := r.resolveConfigValue(clone); err != nil {
		return reflect.Value{}, err
	}

	return clone, nil
}

func (r *Resolver) expandPlaceholders(input string) (string, error) {
	var builder strings.Builder

	for cursor := 0; cursor < len(input); {
		start := strings.Index(input[cursor:], placeholderStart)
		if start == -1 {
			builder.WriteString(input[cursor:])
			return builder.String(), nil
		}

		start += cursor
		builder.WriteString(input[cursor:start])

		// Only the explicit ${VAR} form is supported. We do not inherit broader
		// shell expansion semantics such as bare $VAR or default expressions.
		end := strings.Index(input[start+len(placeholderStart):], placeholderEnd)
		if end == -1 {
			return "", fmt.Errorf("invalid configuration placeholder in %q", input)
		}

		end += start + len(placeholderStart)
		expression := input[start+len(placeholderStart) : end]
		if expression == "" || strings.Contains(expression, placeholderStart) {
			return "", fmt.Errorf("invalid configuration placeholder in %q", input)
		}

		value, err := r.resolveExpression(expression)
		if err != nil {
			return "", err
		}

		builder.WriteString(value)
		cursor = end + 1
	}

	return builder.String(), nil
}

func (r *Resolver) resolveExpression(expression string) (string, error) {
	providerName, key, hasProvider := strings.Cut(expression, ":")
	if !hasProvider {
		return r.environment.Resolve(expression)
	}
	if providerName == "" || key == "" {
		return "", fmt.Errorf("invalid secret reference %q", expression)
	}

	provider, ok := r.providers[providerName]
	if !ok {
		return "", fmt.Errorf("unsupported secret provider %q", providerName)
	}

	return provider.Resolve(key)
}

type envProvider struct{}

func (envProvider) Resolve(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return value, nil
}
