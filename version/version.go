package version

import "fmt"

const (
	// Major is for an API incompatible changes.
	Major = 0
	// Minor is for functionality in a backwards-compatible manner.
	Minor = 2
	// Patch is for backwards-compatible bug fixes.
	Patch = 1
)

// Core is the specification version that the package types support.
var Core = fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)
