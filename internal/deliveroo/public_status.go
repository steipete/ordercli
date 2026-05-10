package deliveroo

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/steipete/ordercli/internal/browserpage"
)

var quantityOnlyRe = regexp.MustCompile(`^\d+\s*x$`)
var readBrowserPageText = browserpage.ReadText

type PublicStatus struct {
	URL              string   `json:"url,omitempty"`
	Title            string   `json:"title,omitempty"`
	Customer         string   `json:"customer,omitempty"`
	EstimatedArrival string   `json:"estimated_arrival,omitempty"`
	Status           string   `json:"status,omitempty"`
	StatusDetail     string   `json:"status_detail,omitempty"`
	Courier          string   `json:"courier,omitempty"`
	Transport        string   `json:"transport,omitempty"`
	Address          string   `json:"address,omitempty"`
	Restaurant       string   `json:"restaurant,omitempty"`
	OrderNumber      string   `json:"order_number,omitempty"`
	Items            []string `json:"items,omitempty"`
	RawText          string   `json:"raw_text,omitempty"`
}

func FetchPublicStatus(ctx context.Context, targetURL string, timeout time.Duration) (PublicStatus, error) {
	res, err := readBrowserPageText(ctx, targetURL, browserpage.Options{
		Timeout:  timeout,
		Headless: true,
	})
	if err != nil {
		return PublicStatus{}, err
	}
	status := ParsePublicStatus(res.Text)
	status.URL = res.FinalURL
	status.Title = res.Title
	status.RawText = res.Text
	return status, nil
}

func ParsePublicStatus(text string) PublicStatus {
	lines := compactLines(text)
	out := PublicStatus{}
	deliveryIdx := indexOf(lines, "Delivery")
	orderDetailsIdx := indexOf(lines, "Order Details")

	for _, line := range lines {
		if strings.HasPrefix(line, "This order is for ") {
			out.Customer = strings.TrimSpace(strings.TrimPrefix(line, "This order is for "))
			break
		}
	}

	if idx := indexOf(lines, "Estimated arrival"); idx != -1 {
		if idx+1 < len(lines) {
			out.EstimatedArrival = lines[idx+1]
		}
		if idx+2 < len(lines) {
			out.Status = lines[idx+2]
		}
		if idx+3 < len(lines) && !isNoiseMarker(lines[idx+3]) {
			out.StatusDetail = lines[idx+3]
		}
	}

	if out.Status == "" {
		if idx := indexOfPrefixed(lines, "This order is for "); idx != -1 {
			end := len(lines)
			if deliveryIdx != -1 {
				end = deliveryIdx
			}
			section := lines[idx+1 : min(idx+3, end)]
			if len(section) > 0 {
				out.Status = section[0]
			}
			if len(section) > 1 && !isNoiseMarker(section[1]) {
				out.StatusDetail = section[1]
			}
		}
	}

	if deliveryIdx != -1 {
		end := len(lines)
		if orderDetailsIdx != -1 {
			end = orderDetailsIdx
		}
		section := lines[deliveryIdx+1 : end]
		switch len(section) {
		case 0:
		case 1:
			out.Address = section[0]
		case 2:
			out.Courier = section[0]
			if looksLikeAddress(section[1]) {
				out.Address = section[1]
			} else {
				out.Transport = section[1]
			}
		default:
			out.Courier = section[0]
			out.Transport = section[1]
			out.Address = section[2]
		}
	}

	if idx := orderDetailsIdx; idx != -1 {
		if idx+1 < len(lines) {
			out.Restaurant = lines[idx+1]
		}
		if idx+2 < len(lines) {
			out.OrderNumber = lines[idx+2]
		}
		for i := idx + 3; i < len(lines); i++ {
			line := lines[i]
			if isFooterMarker(line) {
				break
			}
			if quantityOnlyRe.MatchString(line) {
				if i+1 < len(lines) && !isFooterMarker(lines[i+1]) {
					out.Items = append(out.Items, line+" "+lines[i+1])
					i++
				}
				continue
			}
			if strings.Contains(line, "x ") || strings.Contains(line, "x\t") {
				out.Items = append(out.Items, line)
			}
		}
	}

	return out
}

func compactLines(text string) []string {
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func indexOf(lines []string, want string) int {
	for i, line := range lines {
		if line == want {
			return i
		}
	}
	return -1
}

func indexOfPrefixed(lines []string, prefix string) int {
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func looksLikeAddress(line string) bool {
	line = strings.TrimSpace(line)
	return strings.Contains(line, ",") || strings.ContainsAny(line, "0123456789")
}

func isFooterMarker(line string) bool {
	switch line {
	case "Discover Deliveroo", "Investors", "About us", "Takeaway", "More", "Legal", "Support", "Cookies", "© 2026 Deliveroo":
		return true
	default:
		return false
	}
}

func isNoiseMarker(line string) bool {
	switch line {
	case "Keyboard shortcuts", "Terms", "Report a map error", "Delivery", "Order Details":
		return true
	default:
		return isFooterMarker(line)
	}
}
