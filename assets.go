package assets

import (
	"embed"
	"io/fs"
)

//go:embed static/* templates/*
var files embed.FS

func FS() fs.FS { return files }
