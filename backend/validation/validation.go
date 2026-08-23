package validation

import (
	"encoding/json"
	"errors"
	"food-lab-chain-service/domain"
	"io"
	"net/http"
	"strings"
)

func DecodeHandoff(r *http.Request) (domain.HandoffRequest, error) {
	var input domain.HandoffRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 2048)).Decode(&input); err != nil {
		return input, errors.New("body must be valid JSON")
	}
	input.To = strings.TrimSpace(input.To)
	if input.To == "" || len([]rune(input.To)) > 120 {
		return input, errors.New("recipient must be 1 to 120 characters")
	}
	return input, nil
}
