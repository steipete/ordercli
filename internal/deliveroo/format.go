package deliveroo

import (
	"fmt"
	"strings"
)

func (o Order) Summary() string {
	parts := make([]string, 0, 8)

	if o.ID != "" {
		parts = append(parts, "id="+o.ID)
	}
	if o.OrderNumber != "" {
		parts = append(parts, "number="+o.OrderNumber)
	}
	if o.Status != "" {
		parts = append(parts, "status="+o.Status)
	}
	if o.Restaurant != nil && o.Restaurant.Name != "" {
		parts = append(parts, "restaurant="+o.Restaurant.Name)
	}
	if o.Total != nil {
		if o.CurrencySymbol != "" {
			parts = append(parts, fmt.Sprintf("total=%s%.2f", o.CurrencySymbol, *o.Total))
		} else {
			parts = append(parts, fmt.Sprintf("total=%.2f", *o.Total))
		}
	}
	if o.EstimatedDeliveryAt != "" {
		parts = append(parts, "eta="+o.EstimatedDeliveryAt)
	}
	if o.SubmittedAt != "" {
		parts = append(parts, "submitted_at="+o.SubmittedAt)
	}

	if len(parts) == 0 {
		return "order"
	}
	return strings.Join(parts, " ")
}

func (s PublicStatus) DetailsString() string {
	lines := make([]string, 0, 12+len(s.Items))
	appendLine := func(key, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	appendLine("restaurant", s.Restaurant)
	appendLine("order", s.OrderNumber)
	appendLine("for", s.Customer)
	appendLine("status", s.Status)
	appendLine("detail", s.StatusDetail)
	appendLine("eta", s.EstimatedArrival)
	appendLine("courier", s.Courier)
	appendLine("transport", s.Transport)
	appendLine("address", s.Address)
	appendLine("url", s.URL)
	if len(s.Items) > 0 {
		lines = append(lines, "items:")
		for _, item := range s.Items {
			lines = append(lines, "  "+item)
		}
	}
	return strings.Join(lines, "\n")
}
