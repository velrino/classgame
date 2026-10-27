package pagination

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type Params struct {
	Limit int
	After string
}

func Parse(values url.Values) (Params, error) {
	p := Params{Limit: 50}
	if value := values.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			return p, errors.New("limit must be between 1 and 100")
		}
		p.Limit = limit
	}
	if token := values.Get("cursor"); token != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(decoded) == 0 || len(decoded) > 1500 || strings.ContainsAny(string(decoded), "/\x00") {
			return p, errors.New("invalid pagination cursor")
		}
		p.After = string(decoded)
	}
	return p, nil
}

func Cursor(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id))
}
