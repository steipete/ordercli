package cli

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/steipete/ordercli/internal/foodora"
)

func isInvalidClientErr(err error) bool {
	var he *foodora.HTTPError
	if !errors.As(err, &he) {
		return false
	}
	if he.StatusCode != 400 && he.StatusCode != 401 {
		return false
	}
	var v struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(he.Body, &v) == nil && strings.EqualFold(strings.TrimSpace(v.Error), "invalid_client") {
		return true
	}
	return false
}
