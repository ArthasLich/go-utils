package utils

import "regexp"

var (
	regUID = regexp.MustCompile(`^uid=[0-9]+\(([0-9a-zA-Z_]+)\).*$`)
)
