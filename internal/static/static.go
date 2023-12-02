package static

import (
	"embed"
	"io/fs"
)

//go:embed *
var files embed.FS

func Resources() fs.FS {
	return files
}
