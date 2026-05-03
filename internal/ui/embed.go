package ui

import "embed"

// distFS holds embedded production UI assets.
//
//go:embed dist/*
var distFS embed.FS
