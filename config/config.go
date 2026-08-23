package config

import (
	"os"
	"strconv"
)

func Port() int {
	p, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil || p < 1 || p > 65535 {
		return 8080
	}
	return p
}
