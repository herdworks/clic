package clic

import (
	"time"
)

// ============================ Package Variables ============================
var (
	LibVersion        = "v0.4.2"
	LibReleaseDate, _ = time.Parse(time.RFC3339, "2026-08-13T20:00:00-07:00") // Usage: LibReleaseDate.Format(time.RFC3339)
)
