package web

import "embed"

// Dist is the built frontend (web/dist). `task build` / `task frontend` populates web/dist; a placeholder may be
// present so embed and go build succeed before the first frontend build.
//
//go:embed all:dist
var Dist embed.FS
