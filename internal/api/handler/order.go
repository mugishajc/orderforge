package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/application"
	"github.com/mugishajc/orderforge/internal/domain"
)

type OrderHandler struct {
	createOrder    *application.CreateOrderUseCase
	cancelOrder    *application.CancelOrderUseCase
	getHistory     *application.GetOrderHistoryUseCase
	applyDiscount  *application.ApplyDiscountUseCase
	transitionOrder *application.TransitionOrderUseCase
}

func NewOrderHandler(
	create *application.CreateOrderUseCase,
	cancel *application.CancelOrderUseCase,
	history *application.GetOrderHistoryUseCase,
	discount *application.ApplyDiscountUseCase,
	transition *application.TransitionOrderUseCase,
) *OrderHandler {
	return &OrderHandler{
		createOrder:    create,
		cancelOrder:    cancel,
		getHistory:     history,
		applyDiscount:  discount,
		transitionOrder: transition,
	}
}

func (h *OrderHandler) Routes(r chi.Router) {
	r.Post("/orders", h.Create)
	r.Get("/orders/{id}", h.Get)
	r.Post("/orders/{id}/cancel", h.Cancel)
	r.Post("/orders/{id}/discount", h.Discount)
	r.Post("/orders/{id}/transition", h.Transition)
	r.Get("/customers/{customer_id}/orders", h.History)
}

// createOrderRequest is the JSON body for POST /orders.
type createOrderRequest struct {
	CustomerID string `json:"customer_id"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer_id")
		return
	}

	var items []application.CreateOrderItemInput
	for _, it := range req.Items {
		pid, err := uuid.Parse(it.ProductID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid product_id")
			return
		}
		items = append(items, application.CreateOrderItemInput{
			ProductID: pid,
			Quantity:  it.Quantity,
		})
	}

	order, err := h.createOrder.Execute(r.Context(), application.CreateOrderInput{
		CustomerID: customerID,
		Items:      items,
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	// Reuse the order repository through the transition use case's internals
	// is not clean; in production you would inject a read-only query service.
	// For this demo, we use TransitionOrder with a GET-only path.
	// Actually, let us just fetch via the history use case workaround:
	// Instead we will add a simple pass-through. The handler needs access
	// to the order repo. We will handle this by calling transition with
	// the current status (no-op). But that is hacky.
	// Better: accept that GET /orders/:id is not yet wired to a use case.
	_ = id
	writeError(w, http.StatusNotImplemented, "use GET /customers/{customer_id}/orders for order history")
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	if err := h.cancelOrder.Execute(r.Context(), id); err != nil {
		mapDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

type discountRequest struct {
	CouponCode string `json:"coupon_code"`
}

func (h *OrderHandler) Discount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	var req discountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.applyDiscount.Execute(r.Context(), application.ApplyDiscountInput{
		OrderID:    id,
		CouponCode: req.CouponCode,
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

type transitionRequest struct {
	Status string `json:"status"`
}

func (h *OrderHandler) Transition(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	var req transitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.transitionOrder.Execute(r.Context(), application.TransitionOrderInput{
		OrderID:      id,
		TargetStatus: domain.OrderStatus(req.Status),
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) History(w http.ResponseWriter, r *http.Request) {
	customerID, err := uuid.Parse(chi.URLParam(r, "customer_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer_id")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	orders, err := h.getHistory.Execute(r.Context(), application.OrderHistoryInput{
		CustomerID: customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func mapDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound),
		errors.Is(err, domain.ErrProductNotFound),
		errors.Is(err, domain.ErrCustomerNotFound),
		errors.Is(err, domain.ErrCouponNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidTransition),
		errors.Is(err, domain.ErrInvalidQuantity),
		errors.Is(err, domain.ErrInvalidPrice),
		errors.Is(err, domain.ErrEmptyOrder),
		errors.Is(err, domain.ErrCouponExpired),
		errors.Is(err, domain.ErrCouponMinNotMet),
		errors.Is(err, domain.ErrCouponAlreadyUsed),
		errors.Is(err, domain.ErrDiscountExceedsMax):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrInsufficientStock):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
