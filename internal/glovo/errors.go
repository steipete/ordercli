package glovo

import "fmt"

// HTTPError represents an HTTP error response from the Glovo API.
type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	body := string(e.Body)
	if len(body) > 300 {
		body = body[:300] + "..."
	}
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.URL, e.StatusCode, body)
}

// IsUnauthorized returns true if the error is an authentication error.
func (e *HTTPError) IsUnauthorized() bool {
	return e.StatusCode == 401 || e.StatusCode == 403
}
