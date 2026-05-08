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

type OrderCreatedEvent struct {
	EventType       string      `json:"eventType"`
	Version         string      `json:"version"`
	OrderID         string      `json:"orderId"`
	CustomerID      string      `json:"customerId"`
	Items           []OrderItem `json:"items"`
	SimulateFailure bool        `json:"simulateFailure"`
	Meta            EventMeta   `json:"_meta"`
}

type PaymentResultEvent struct {
	EventType     string      `json:"eventType"`
	Version       string      `json:"version"`
	PaymentID     string      `json:"paymentId"`
	OrderID       string      `json:"orderId"`
	CustomerID    string      `json:"customerId"`
	Items         []OrderItem `json:"items"`
	TotalAmount   float64     `json:"totalAmount"`
	Status        string      `json:"status"`
	FailureReason string      `json:"failureReason,omitempty"`
	Meta          EventMeta   `json:"_meta"`
}

type Payment struct {
	PaymentID     string  `dynamodbav:"paymentId"`
	OrderID       string  `dynamodbav:"orderId"`
	CustomerID    string  `dynamodbav:"customerId"`
	TotalAmount   float64 `dynamodbav:"totalAmount"`
	Status        string  `dynamodbav:"status"`
	FailureReason string  `dynamodbav:"failureReason,omitempty"`
	CorrelationID string  `dynamodbav:"correlationId"`
	CreatedAt     string  `dynamodbav:"createdAt"`
}
