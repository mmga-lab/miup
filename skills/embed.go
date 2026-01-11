package skills

import "embed"

// MiupSkill contains the embedded skill files for miup.
//
//go:embed miup/*
//go:embed miup/references/*
var MiupSkill embed.FS
