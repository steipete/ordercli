package deliveroo

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type OrdersResponse struct {
	Orders []Order `json:"orders"`
	Count  int     `json:"count"`
}

type StringNumber string

func (v *StringNumber) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*v = ""
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*v = StringNumber(text)
		return nil
	}

	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err == nil {
		*v = StringNumber(number.String())
		return nil
	}

	return fmt.Errorf("unsupported string/number value %s", data)
}

type Order struct {
	ID                  string       `json:"id"`
	OrderNumber         StringNumber `json:"order_number"`
	Status              string       `json:"status"`
	StatusTimestamp     string       `json:"status_timestamp"`
	OrderType           string       `json:"order_type"`
	PaymentStatus       string       `json:"payment_status"`
	EstimatedDeliveryAt string       `json:"estimated_delivery_at"`
	DeliveredAt         string       `json:"delivered_at"`
	SubmittedAt         string       `json:"submitted_at"`
	Total               *float64     `json:"total"`
	OriginalTotal       *float64     `json:"original_total"`
	CurrencySymbol      string       `json:"currency_symbol"`
	CurrencyCode        string       `json:"currency_code"`
	Restaurant          *Restaurant  `json:"restaurant"`
}

type Restaurant struct {
	Name string `json:"name"`
}
