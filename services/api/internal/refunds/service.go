package refunds

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-platform/services/api/internal/commerce"
	"github.com/yxshee/marketplace-platform/services/api/internal/platform/identifier"
)

const (
	RequestStatusPending  = "pending"
	RequestStatusApproved = "approved"
	RequestStatusRejected = "rejected"

	DecisionApprove = "approve"
	DecisionReject  = "reject"
)

var (
	ErrInvalidOrder           = errors.New("order is required")
	ErrInvalidVendor          = errors.New("vendor is required")
	ErrInvalidShipment        = errors.New("shipment is required")
	ErrInvalidReason          = errors.New("reason is required")
	ErrInvalidAmount          = errors.New("requested amount is invalid")
	ErrOrderNotRefundable     = errors.New("order status does not allow refunds")
	ErrShipmentNotFound       = errors.New("shipment not found")
	ErrRefundRequestDuplicate = errors.New("pending refund request already exists")
	ErrRefundRequestNotFound  = errors.New("refund request not found")
	ErrRefundRequestForbidden = errors.New("refund request forbidden")
	ErrInvalidStatusFilter    = errors.New("status filter is invalid")
	ErrInvalidDecision        = errors.New("decision is invalid")
	ErrDecisionConflict       = errors.New("refund request decision already made")
)

// RefundRequest captures buyer-initiated refund intent and vendor decision outcome.
type RefundRequest struct {
	ID                   string     `json:"id"`
	OrderID              string     `json:"order_id"`
	ShipmentID           string     `json:"shipment_id"`
	VendorID             string     `json:"vendor_id"`
	BuyerUserID          string     `json:"buyer_user_id,omitempty"`
	GuestToken           string     `json:"guest_token,omitempty"`
	Reason               string     `json:"reason"`
	RequestedAmountCents int64      `json:"requested_amount_cents"`
	Currency             string     `json:"currency"`
	Status               string     `json:"status"`
	Outcome              string     `json:"outcome"`
	Decision             string     `json:"decision,omitempty"`
	DecisionReason       string     `json:"decision_reason,omitempty"`
	DecidedByUserID      string     `json:"decided_by_user_id,omitempty"`
	DecidedAt            *time.Time `json:"decided_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Service stores refund requests in-memory for API workflow validation.
type Service struct {
	mu                     sync.Mutex
	requestsByID           map[string]RefundRequest
	requestIDsByVendorID   map[string][]string
	pendingByOrderShipment map[string]string
}

func NewService() *Service {
	return &Service{
		requestsByID:           make(map[string]RefundRequest),
		requestIDsByVendorID:   make(map[string][]string),
		pendingByOrderShipment: make(map[string]string),
	}
}

// CreateRequest creates a refund request for a buyer-owned order shipment.
func (s *Service) CreateRequest(
	actor commerce.Actor,
	order commerce.Order,
	shipmentID string,
	reason string,
	requestedAmountCents int64,
) (RefundRequest, error) {
	if strings.TrimSpace(order.ID) == "" {
		return RefundRequest{}, ErrInvalidOrder
	}
	if !isRefundableOrderStatus(order.Status) {
		return RefundRequest{}, ErrOrderNotRefundable
	}

	normalizedShipmentID := strings.TrimSpace(shipmentID)
	if normalizedShipmentID == "" {
		return RefundRequest{}, ErrInvalidShipment
	}

	normalizedReason := strings.TrimSpace(reason)
	if normalizedReason == "" {
		return RefundRequest{}, ErrInvalidReason
	}

	shipment, found := findOrderShipment(order, normalizedShipmentID)
	if !found {
		return RefundRequest{}, ErrShipmentNotFound
	}

	targetAmount := requestedAmountCents
	if targetAmount == 0 {
		targetAmount = shipment.TotalCents
	}
	if targetAmount <= 0 || targetAmount > shipment.TotalCents {
		return RefundRequest{}, ErrInvalidAmount
	}

	now := time.Now().UTC()
	request := RefundRequest{
		ID:                   identifier.New("rfr"),
		OrderID:              order.ID,
		ShipmentID:           shipment.ID,
		VendorID:             shipment.VendorID,
		BuyerUserID:          strings.TrimSpace(actor.BuyerUserID),
		GuestToken:           strings.TrimSpace(actor.GuestToken),
		Reason:               normalizedReason,
		RequestedAmountCents: targetAmount,
		Currency:             order.Currency,
		Status:               RequestStatusPending,
		Outcome:              RequestStatusPending,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pendingKey := makePendingKey(order.ID, shipment.ID)
	if _, exists := s.pendingByOrderShipment[pendingKey]; exists {
		return RefundRequest{}, ErrRefundRequestDuplicate
	}

	s.requestsByID[request.ID] = request
	s.requestIDsByVendorID[request.VendorID] = append(s.requestIDsByVendorID[request.VendorID], request.ID)
	s.pendingByOrderShipment[pendingKey] = request.ID

	return request, nil
}

// ListVendorRequests returns refund requests owned by a vendor, optionally filtered by status.
func (s *Service) ListVendorRequests(vendorID, statusFilter string) ([]RefundRequest, error) {
	normalizedVendorID := strings.TrimSpace(vendorID)
	if normalizedVendorID == "" {
		return nil, ErrInvalidVendor
	}

	normalizedStatusFilter := normalizeStatus(statusFilter)
	if normalizedStatusFilter != "" && !isValidStatus(normalizedStatusFilter) {
		return nil, ErrInvalidStatusFilter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := s.requestIDsByVendorID[normalizedVendorID]
	result := make([]RefundRequest, 0, len(ids))
	for _, id := range ids {
		request, exists := s.requestsByID[id]
		if !exists {
			continue
		}
		if normalizedStatusFilter != "" && request.Status != normalizedStatusFilter {
			continue
		}
		result = append(result, request)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result, nil
}

// DecideRequest applies a vendor decision to a pending refund request.
func (s *Service) DecideRequest(vendorID, requestID, decision, decisionReason, actorUserID string) (RefundRequest, error) {
	normalizedVendorID := strings.TrimSpace(vendorID)
	if normalizedVendorID == "" {
		return RefundRequest{}, ErrInvalidVendor
	}

	normalizedRequestID := strings.TrimSpace(requestID)
	if normalizedRequestID == "" {
		return RefundRequest{}, ErrRefundRequestNotFound
	}

	normalizedDecision := normalizeDecision(decision)
	if normalizedDecision != DecisionApprove && normalizedDecision != DecisionReject {
		return RefundRequest{}, ErrInvalidDecision
	}

	normalizedDecisionReason := strings.TrimSpace(decisionReason)

	s.mu.Lock()
	defer s.mu.Unlock()

	request, exists := s.requestsByID[normalizedRequestID]
	if !exists {
		return RefundRequest{}, ErrRefundRequestNotFound
	}
	if request.VendorID != normalizedVendorID {
		return RefundRequest{}, ErrRefundRequestForbidden
	}
	if request.Status != RequestStatusPending {
		return RefundRequest{}, ErrDecisionConflict
	}

	now := time.Now().UTC()
	request.Decision = normalizedDecision
	request.DecisionReason = normalizedDecisionReason
	request.DecidedByUserID = strings.TrimSpace(actorUserID)
	request.DecidedAt = &now
	request.UpdatedAt = now

	if normalizedDecision == DecisionApprove {
		request.Status = RequestStatusApproved
		request.Outcome = RequestStatusApproved
	} else {
		request.Status = RequestStatusRejected
		request.Outcome = RequestStatusRejected
	}

	s.requestsByID[normalizedRequestID] = request
	delete(s.pendingByOrderShipment, makePendingKey(request.OrderID, request.ShipmentID))

	return request, nil
}

func isRefundableOrderStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case commerce.OrderStatusPaid, commerce.OrderStatusCODConfirmed:
		return true
	default:
		return false
	}
}

func findOrderShipment(order commerce.Order, shipmentID string) (commerce.OrderShipment, bool) {
	for _, shipment := range order.Shipments {
		if shipment.ID == shipmentID {
			return shipment, true
		}
	}
	return commerce.OrderShipment{}, false
}

func makePendingKey(orderID, shipmentID string) string {
	return orderID + ":" + shipmentID
}

func normalizeDecision(decision string) string {
	return strings.ToLower(strings.TrimSpace(decision))
}

func normalizeStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func isValidStatus(status string) bool {
	switch normalizeStatus(status) {
	case RequestStatusPending, RequestStatusApproved, RequestStatusRejected:
		return true
	default:
		return false
	}
}
