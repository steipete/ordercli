package deliveroo

type OrdersResponse struct {
	Orders []Order `json:"orders"`
	Count  int     `json:"count"`
}

type Order struct {
	ID                  string      `json:"id"`
	OrderNumber         string      `json:"order_number"`
	Status              string      `json:"status"`
	StatusTimestamp     string      `json:"status_timestamp"`
	OrderType           string      `json:"order_type"`
	PaymentStatus       string      `json:"payment_status"`
	EstimatedDeliveryAt string      `json:"estimated_delivery_at"`
	DeliveredAt         string      `json:"delivered_at"`
	SubmittedAt         string      `json:"submitted_at"`
	Total               *float64    `json:"total"`
	OriginalTotal       *float64    `json:"original_total"`
	CurrencySymbol      string      `json:"currency_symbol"`
	CurrencyCode        string      `json:"currency_code"`
	Restaurant          *Restaurant `json:"restaurant"`
}

type Restaurant struct {
	Name string `json:"name"`
}
