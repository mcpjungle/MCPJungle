// Package accesstoken provides shared logic to resolve effective access tokens
// from config inputs that may define either:
// - an inline access token,
// - a reference to an environment variable,
// - or a reference to a file.
//
// It centralizes resolution precedence and validation so CLI and services
// behave consistently when processing access token configuration.
package accesstoken
