package cmd

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// extractErrorInfo attempts to extract status code and body from various error types
func extractErrorInfo(err error) (int, string) {
	// Check if error contains status code in the message
	statusRegex := regexp.MustCompile(`status(?:\s+code)?:?\s*(\d+)`)
	match := statusRegex.FindStringSubmatch(err.Error())
	if len(match) > 1 {
		if statusCode, parseErr := strconv.Atoi(match[1]); parseErr == nil {
			return statusCode, ""
		}
	}

	// Look for JSON response in the error
	jsonStart := strings.Index(err.Error(), "{")
	jsonEnd := strings.LastIndex(err.Error(), "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonBody := err.Error()[jsonStart : jsonEnd+1]
		return 0, jsonBody
	}

	// Check if error message contains specific keywords
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "400") || strings.Contains(errMsg, "bad request") {
		return http.StatusBadRequest, ""
	}
	if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized") {
		return http.StatusUnauthorized, ""
	}
	if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "forbidden") {
		return http.StatusForbidden, ""
	}
	if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") {
		return http.StatusNotFound, ""
	}
	if strings.Contains(errMsg, "409") || strings.Contains(errMsg, "conflict") ||
		strings.Contains(errMsg, "already exists") {
		return http.StatusConflict, ""
	}
	if strings.Contains(errMsg, "500") || strings.Contains(errMsg, "server error") {
		return http.StatusInternalServerError, ""
	}

	// No status code found
	return 0, ""
}
