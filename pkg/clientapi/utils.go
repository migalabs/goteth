package clientapi

import "strings"

const (
	missingData = "404"
)

func response404(err string) bool {
	return strings.Contains(err, missingData)
}
