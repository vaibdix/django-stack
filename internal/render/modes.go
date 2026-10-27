package render

import (
	"io/fs"
)

const (
	FileMode   fs.FileMode = 0o644
	secretMode fs.FileMode = 0o600
	execMode   fs.FileMode = 0o755
	dirMode    fs.FileMode = 0o755
)
