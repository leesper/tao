package tao

import (
	"strconv"
	"strings"
)

func parseAddress(address string) (string, int, error) {
	parts := strings.Split(address, ":")
	port, err := strconv.Atoi(parts[1])
	return parts[0], port, err
}
