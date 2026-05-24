package version

import (
	"runtime/debug"
	"strconv"
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

func CompareVersions(a, b string) int {
	na := normalizeVersion(a)
	nb := normalizeVersion(b)

	if na == "" && nb == "" {
		return 0
	}
	if na == "" {
		return -1
	}
	if nb == "" {
		return 1
	}

	aparts := strings.Split(na, ".")
	bparts := strings.Split(nb, ".")

	maxLen := len(aparts)
	if len(bparts) > maxLen {
		maxLen = len(bparts)
	}

	for i := range maxLen {
		var ai, bi int
		if i < len(aparts) {
			ai, _ = strconv.Atoi(aparts[i])
		}
		if i < len(bparts) {
			bi, _ = strconv.Atoi(bparts[i])
		}

		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}

	return 0
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}

	return v
}
