package router

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func decodeJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func bearerToken(headerValue string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(headerValue), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("missing bearer token")
	}
	return strings.TrimSpace(parts[1]), nil
}

func parsePagination(r *http.Request, defaultLimit, maxLimit int) (int, int, error) {
	limit := defaultLimit
	limitRaw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed < 1 || parsed > maxLimit {
			return 0, 0, errors.New("limit must be between 1 and " + strconv.Itoa(maxLimit))
		}
		limit = parsed
	}

	offset := 0
	offsetRaw := strings.TrimSpace(r.URL.Query().Get("offset"))
	if offsetRaw != "" {
		parsed, err := strconv.Atoi(offsetRaw)
		if err != nil || parsed < 0 {
			return 0, 0, errors.New("offset must be zero or positive")
		}
		offset = parsed
	}

	return limit, offset, nil
}

func paginate(total, limit, offset int) (int, int) {
	if total < 0 {
		total = 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}

	end := offset + limit
	if end > total {
		end = total
	}
	return offset, end
}
