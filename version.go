package clif

import (
	"time"
)

// ============================ Package Variables ============================
var (
	LibVersion        = "v0.1.0"
	LibReleaseDate, _ = time.Parse(time.RFC3339, "2025-08-31T20:47:36-07:00") // Usage: LibReleaseDate.Format(time.RFC3339)
)
