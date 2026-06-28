//go:build !darwin

package notification

import (
	_ "embed"
)

//go:embed giis-code-icon-solo.png
var Icon []byte
