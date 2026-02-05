package version

import (
	"runtime/debug"
	"strings"
)

var Version = "dev"

func GetVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		version := info.Main.Version
		if version != "" && version != "(devel)" {
			return strings.ReplaceAll(version, "+dirty", "")
		}
	}

	return Version
}
