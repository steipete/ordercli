package deliveroo

import (
	"fmt"
	"strconv"
	"strings"
)

func (o Order) Summary() string {
	parts := make([]string, 0, 8)

	if o.ID != "" {
		parts = append(parts, "id="+o.ID)
	}
	if o.OrderNumber != 0 {
		parts = append(parts, "number="+strconv.Itoa(o.OrderNumber))
	}
	if o.Status != "" {
		parts = append(parts, "status="+o.Status)
	}
	if o.Restaurant != nil && o.Restaurant.Name != "" {
		parts = append(parts, "restaurant="+o.Restaurant.Name)
	}
	if o.Total != "" {
		if o.CurrencySymbol != "" {
			parts = append(parts, fmt.Sprintf("total=%s%s", o.CurrencySymbol, o.Total))
		} else {
			parts = append(parts, "total="+o.Total)
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
