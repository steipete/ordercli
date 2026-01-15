package glovo

// OrdersResponse represents the response from /v3/customer/orders-list
type OrdersResponse struct {
	Pagination Pagination `json:"pagination"`
	Orders     []Order    `json:"orders"`
	Rows       any        `json:"rows"`
}

// Pagination contains pagination info for order list
type Pagination struct {
	CurrentLimit int     `json:"currentLimit"`
	Next         *string `json:"next"`
}

// Order represents a single order in the history
type Order struct {
	OrderID                   int     `json:"orderId"`
	OrderURN                  string  `json:"orderUrn"`
	Image                     Image   `json:"image"`
	Content                   Content `json:"content"`
	Footer                    Footer  `json:"footer"`
	Style                     string  `json:"style"`
	LayoutType                string  `json:"layoutType"`
	IsNewOrderTrackingEnabled bool    `json:"isNewOrderTrackingEnabled"`
	CourierName               *string `json:"courierName"`
}

// Image holds light/dark mode image IDs
type Image struct {
	LightImageID string `json:"lightImageId"`
	DarkImageID  string `json:"darkImageId"`
}

// Content contains order display content
type Content struct {
	Title string        `json:"title"`
	Body  []ContentBody `json:"body"`
}

// ContentBody is a single content block
type ContentBody struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// Footer contains order footer info (price, status)
type Footer struct {
	Left  *FooterItem `json:"left"`
	Right *FooterItem `json:"right"`
}

// FooterItem is a footer element
type FooterItem struct {
	Type string `json:"type"`
	Data any    `json:"data"` // Can be string or object (button)
}

// DataString returns Data as string if it is one, otherwise empty
func (f *FooterItem) DataString() string {
	if s, ok := f.Data.(string); ok {
		return s
	}
	return ""
}

// BasketsResponse represents the response from /v1/authenticated/customers/{id}/baskets
type BasketsResponse []Basket

// Basket represents a shopping cart for a store
type Basket struct {
	StoreID          int           `json:"storeId"`
	StoreAddressID   int           `json:"storeAddressId"`
	StoreName        string        `json:"storeName"`
	StoreSlug        string        `json:"storeSlug"`
	Products         []BasketItem  `json:"products"`
	SubTotal         float64       `json:"subTotal"`
	DeliveryFee      float64       `json:"deliveryFee"`
	ServiceFee       float64       `json:"serviceFee"`
	SmallOrderFee    float64       `json:"smallOrderFee"`
	Total            float64       `json:"total"`
	Currency         string        `json:"currency"`
	MinOrderValue    float64       `json:"minOrderValue"`
	IsMinOrderMet    bool          `json:"isMinOrderMet"`
}

// BasketItem represents an item in the cart
type BasketItem struct {
	ID          int     `json:"id"`
	ProductID   int     `json:"productId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	TotalPrice  float64 `json:"totalPrice"`
}

// UserResponse represents the response from /v3/me
type UserResponse struct {
	ID                    int          `json:"id"`
	Type                  string       `json:"type"`
	URN                   string       `json:"urn"`
	Name                  string       `json:"name"`
	Picture               *string      `json:"picture"`
	Email                 string       `json:"email"`
	Description           *string      `json:"description"`
	FacebookID            *string      `json:"facebookId"`
	PreferredCityCode     string       `json:"preferredCityCode"`
	PreferredLanguage     string       `json:"preferredLanguage"`
	PreferredLanguageRegion string     `json:"preferredLanguageRegion"`
	Locale                string       `json:"locale"`
	DeviceURN             *string      `json:"deviceUrn"`
	AnalyticsID           *string      `json:"analyticsId"`
	MediaCampaign         *string      `json:"mediaCampaign"`
	MediaSource           *string      `json:"mediaSource"`
	OS                    *string      `json:"os"`
	DeliveredOrdersCount  int          `json:"deliveredOrdersCount"`
	PhoneNumber           *PhoneNumber `json:"phoneNumber"`
	CompanyDetail         *string      `json:"companyDetail"`
	VirtualBalance        *Balance     `json:"virtualBalance"`
	FreeOrders            int          `json:"freeOrders"`
	PaymentMethod         string       `json:"paymentMethod"`
	PaymentWay            string       `json:"paymentWay"`
	CurrentCard           *string      `json:"currentCard"`
	AccumulatedDebt       float64      `json:"accumulatedDebt"`
	Defaulter             bool         `json:"defaulter"`
	Gender                *string      `json:"gender"`
	AgeMin                *int         `json:"ageMin"`
	AgeMax                *int         `json:"ageMax"`
	Birthday              *string      `json:"birthday"`
	PrivacySettings       any          `json:"privacySettings"`
	Permissions           []Permission `json:"permissions"`
	DataPrivacyEnabled    bool         `json:"dataPrivacyEnabled"`
}

// PhoneNumber represents a phone number with country code
type PhoneNumber struct {
	Number      string  `json:"number"`
	CountryCode *string `json:"countryCode"`
}

// Balance represents virtual balance
type Balance struct {
	Balance float64 `json:"balance"`
}

// Permission represents a user permission setting
type Permission struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Enabled bool   `json:"enabled"`
}
