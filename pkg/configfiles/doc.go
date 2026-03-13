// Package configfiles provides shared helpers to load and parse JSON entity
// configuration files used by both CLI commands and config sync reconciliation.
//
// It centralizes common concerns such as:
// - reading and unmarshaling JSON config files,
// - loading desired state from config directories,
// - validating entity names for desired entries,
// - computing content hashes for change detection,
// - and reporting duplicate/blocked config files consistently.
package configfiles
