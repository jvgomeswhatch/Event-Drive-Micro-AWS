package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/platform/order-service/internal/domain"
	"github.com/platform/order-service/internal/idempotency"
	"github.com/platform/order-service/internal/logger"
	"github.com/platform/order-service/internal/repository"
	"github.com/platform/order-service/internal/tracing"
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Publisher interface {
	Publish(ctx context.Context, event any, correlationID, eventType string) error
}

type OrderHandler struct {
	repo      *repository.OrderRepository
	publisher Publisher
	idempot   *idempotency.Client
}

func NewOrderHandler(
	repo *repository.OrderRepository,
	pub Publisher,
	idempot *idempotency.Client,
) *OrderHandler {
	return &OrderHandler{repo: repo, publisher: pub, idempot: idempot}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.Start(r.Context(), "order.create")
	defer span.Finish(nil)
	correlationID := span.TraceID
	span.Tag("correlationId", correlationID)

	var req domain.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateCreateRequest(req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Idempotency
	idempotKey := r.Header.Get("X-Idempotency-Key")
	if idempotKey != "" {
		claimed, err := h.idempot.ClaimKey(ctx, "order-service#"+idempotKey)
		if err != nil {
			logger.Error("Idempotency claim failed", logger.Fields{"correlationId": correlationID, "error": err.Error()})
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !claimed {
			jsonResponse(w, map[string]any{"message": "duplicate request", "cached": true}, http.StatusOK)
			return
		}
	}

	orderID := uuid.New().String()
	now := time.Now().UTC()

	order := &domain.Order{
		OrderID:         orderID,
		CustomerID:      req.CustomerID,
		Items:           req.Items,
		Status:          domain.StatusPending,
		CorrelationID:   correlationID,
		SimulateFailure: req.SimulateFailure,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.repo.Create(ctx, order); err != nil {
		logger.Error("Failed to persist order", logger.Fields{"correlationId": correlationID, "error": err.Error()})
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	event := domain.OrderCreatedEvent{
		EventType:       "order.created",
		Version:         "1",
		OrderID:         orderID,
		CustomerID:      req.CustomerID,
		Items:           req.Items,
		SimulateFailure: req.SimulateFailure,
		Meta: domain.EventMeta{
			CorrelationID: correlationID,
			PublishedAt:   now.Format(time.RFC3339),
			Publisher:     "order-service",
		},
	}

	if err := h.publisher.Publish(ctx, event, correlationID, "order.created"); err != nil {
		logger.Error("Failed to publish order event", logger.Fields{"correlationId": correlationID, "error": err.Error()})
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if idempotKey != "" {
		_ = h.idempot.ResolveKey(ctx, "order-service#"+idempotKey, map[string]string{
			"orderId": orderID,
			"status":  "pending",
		})
	}

	logger.Info("Order created", logger.Fields{
		"correlationId": correlationID,
		"orderId":       orderID,
		"customerId":    req.CustomerID,
		"itemCount":     strconv.Itoa(len(req.Items)),
	})

	w.Header().Set("X-Correlation-ID", correlationID)
	jsonResponse(w, map[string]any{
		"orderId":       orderID,
		"status":        "pending",
		"correlationId": correlationID,
	}, http.StatusAccepted)
}

func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := tracing.FromContext(ctx).TraceID
	orderID := chi.URLParam(r, "orderId")

	if !uuidRe.MatchString(orderID) {
		jsonError(w, "Invalid orderId format", http.StatusBadRequest)
		return
	}

	order, err := h.repo.GetByID(ctx, orderID)
	if err != nil {
		logger.Error("Failed to fetch order", logger.Fields{"correlationId": correlationID, "error": err.Error()})
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if order == nil {
		jsonError(w, "Order not found", http.StatusNotFound)
		return
	}

	jsonResponse(w, order, http.StatusOK)
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := tracing.FromContext(ctx).TraceID
	customerID := r.URL.Query().Get("customerId")

	if customerID == "" {
		jsonError(w, "customerId query param required", http.StatusBadRequest)
		return
	}

	orders, err := h.repo.ListByCustomer(ctx, customerID, 20)
	if err != nil {
		logger.Error("Failed to list orders", logger.Fields{"correlationId": correlationID, "error": err.Error()})
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]any{"orders": orders}, http.StatusOK)
}

func validateCreateRequest(req domain.CreateOrderRequest) error {
	if strings.TrimSpace(req.CustomerID) == "" {
		return fmt.Errorf("customerId is required")
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("items must not be empty")
	}
	if len(req.Items) > 50 {
		return fmt.Errorf("items must not exceed 50")
	}
	for _, item := range req.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			return fmt.Errorf("item productId is required")
		}
		if item.Quantity < 1 || item.Quantity > 1000 {
			return fmt.Errorf("item quantity must be between 1 and 1000")
		}
	}
	return nil
}

func jsonResponse(w http.ResponseWriter, body any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResponse(w, map[string]string{"error": msg}, status)
}
