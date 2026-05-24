package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func pathID(r *http.Request, prefix string) (int, error) {
	path := strings.TrimPrefix(r.URL.Path, prefix)

	segment := strings.SplitN(path, "/", 2)[0]

	id, err := strconv.Atoi(segment)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id %q", segment)
	}

	return id, nil
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return val
}

func queryBool(r *http.Request, key string, defaultVal bool) bool {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultVal
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return defaultVal
	}
}

func queryString(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func queryIntPtr(r *http.Request, key string) (*int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val <= 0 {
		return nil, fmt.Errorf("invalid %s: must be a positive integer", key)
	}
	return &val, nil
}