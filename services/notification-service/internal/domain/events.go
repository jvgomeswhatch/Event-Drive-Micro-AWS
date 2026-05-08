package domain

type EventMeta struct {
	CorrelationID string `json:"correlationId"`
	PublishedAt   string `json:"publishedAt"`
	Publisher     string `json:"publisher"`
}

type IncomingEvent struct {
	EventType     string    `json:"eventType"`
	OrderID       string    `json:"orderId"`
	PaymentID     string    `json:"paymentId,omitempty"`
	ReservationID string    `json:"reservationId,omitempty"`
	TotalAmount   float64   `json:"totalAmount,omitempty"`
	FailureReason string    `json:"failureReason,omitempty"`
	Status        string    `json:"status,omitempty"`
	Meta          EventMeta `json:"_meta"`
}

var StatusMap = map[string]string{
	"payment.succeeded":  "payment_confirmed",
	"payment.failed":     "payment_failed",
	"inventory.reserved": "confirmed",
	"inventory.failed":   "fulfillment_failed",
}

var RelevantEvents = map[string]bool{
	"payment.succeeded":  true,
	"payment.failed":     true,
	"inventory.reserved": true,
	"inventory.failed":   true,
}
