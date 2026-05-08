package domain

type EventMeta struct {
	CorrelationID string `json:"correlationId"`
	PublishedAt   string `json:"publishedAt"`
	Publisher     string `json:"publisher"`
}

type OrderItem struct {
	ProductID string `json:"productId" dynamodbav:"productId"`
	Quantity  int    `json:"quantity"  dynamodbav:"quantity"`
}

type PaymentSucceededEvent struct {
	EventType   string      `json:"eventType"`
	Version     string      `json:"version"`
	PaymentID   string      `json:"paymentId"`
	OrderID     string      `json:"orderId"`
	CustomerID  string      `json:"customerId"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"totalAmount"`
	Status      string      `json:"status"`
	Meta        EventMeta   `json:"_meta"`
}

type InventoryResultEvent struct {
	EventType     string      `json:"eventType"`
	Version       string      `json:"version"`
	ReservationID string      `json:"reservationId"`
	OrderID       string      `json:"orderId"`
	CustomerID    string      `json:"customerId"`
	Items         []OrderItem `json:"items"`
	Status        string      `json:"status"`
	FailureReason string      `json:"failureReason,omitempty"`
	Meta          EventMeta   `json:"_meta"`
}
