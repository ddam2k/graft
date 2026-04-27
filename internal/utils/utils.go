package utils

import (
	"strings"
)

func ParseRepositoryUrl(url string) (repository string, tag string) {
	idx := strings.LastIndex(url, ":")
	if idx == -1 {
		return url, "latest"
	}
	return url[:idx], url[idx+1:]
}
