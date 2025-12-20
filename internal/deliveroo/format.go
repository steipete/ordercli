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
