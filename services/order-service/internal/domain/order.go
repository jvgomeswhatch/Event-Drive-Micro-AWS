package domain

import "time"

type OrderStatus string

const (
	StatusPending           OrderStatus = "pending"
	StatusPaymentConfirmed  OrderStatus = "payment_confirmed"
	StatusPaymentFailed     OrderStatus = "payment_failed"
	StatusConfirmed         OrderStatus = "confirmed"
	StatusFulfillmentFailed OrderStatus = "fulfillment_failed"
)

type OrderItem struct {
	ProductID string `json:"productId" dynamodbav:"productId"`
	Quantity  int    `json:"quantity"  dynamodbav:"quantity"`
}

type Order struct {
	OrderID         string      `json:"orderId"          dynamodbav:"orderId"`
	CustomerID      string      `json:"customerId"       dynamodbav:"customerId"`
	Items           []OrderItem `json:"items"            dynamodbav:"items"`
	Status          OrderStatus `json:"status"           dynamodbav:"status"`
	CorrelationID   string      `json:"correlationId"    dynamodbav:"correlationId"`
	SimulateFailure bool        `json:"simulateFailure"  dynamodbav:"simulateFailure"`
	CreatedAt       time.Time   `json:"createdAt"        dynamodbav:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"        dynamodbav:"updatedAt"`
}

type CreateOrderRequest struct {
	CustomerID      string      `json:"customerId"`
	Items           []OrderItem `json:"items"`
	SimulateFailure bool        `json:"simulateFailure"`
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

type EventMeta struct {
	CorrelationID string `json:"correlationId"`
	PublishedAt   string `json:"publishedAt"`
	Publisher     string `json:"publisher"`
}
