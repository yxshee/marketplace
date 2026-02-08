package commerce

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
)

const (
	DefaultCurrency           = "USD"
	DefaultShippingFeeCents   = int64(500)
	OrderStatusPendingPayment = "pending_payment"
	OrderStatusPaid           = "paid"
	OrderStatusPaymentFailed  = "payment_failed"
	ShipmentStatusPending     = "pending"
)

var (
	ErrInvalidActor      = errors.New("actor is required")
	ErrInvalidProduct    = errors.New("product is invalid")
	ErrInvalidQuantity   = errors.New("quantity must be positive")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrCartItemNotFound  = errors.New("cart item not found")
	ErrCartEmpty         = errors.New("cart is empty")
	ErrCurrencyMismatch  = errors.New("currency mismatch in cart")
	ErrIdempotencyKey    = errors.New("idempotency key is required")
)

// Actor represents the buyer context for cart and checkout operations.
type Actor struct {
	BuyerUserID string `json:"buyer_user_id,omitempty"`
	GuestToken  string `json:"guest_token,omitempty"`
}

func (a Actor) IsGuest() bool {
	return strings.TrimSpace(a.BuyerUserID) == "" && strings.TrimSpace(a.GuestToken) != ""
}

func (a Actor) key() (string, error) {
	if userID := strings.TrimSpace(a.BuyerUserID); userID != "" {
		return "usr:" + userID, nil
	}
	if guestToken := strings.TrimSpace(a.GuestToken); guestToken != "" {
		return "gst:" + guestToken, nil
	}
	return "", ErrInvalidActor
}

// ProductSnapshot contains the checkout-relevant product values captured at add-to-cart time.
type ProductSnapshot struct {
	ID                    string
	VendorID              string
	Title                 string
	Currency              string
	UnitPriceInclTaxCents int64
	StockQty              int32
}

// CartItem is a cart line snapshot.
type CartItem struct {
	ID              string `json:"id"`
	ProductID       string `json:"product_id"`
	VendorID        string `json:"vendor_id"`
	Title           string `json:"title"`
	Qty             int32  `json:"qty"`
	UnitPriceCents  int64  `json:"unit_price_cents"`
	LineTotalCents  int64  `json:"line_total_cents"`
	Currency        string `json:"currency"`
	AvailableStock  int32  `json:"available_stock"`
	LastUpdatedUnix int64  `json:"last_updated_unix"`
}

// Cart is an actor-scoped shopping cart.
type Cart struct {
	ID            string     `json:"id"`
	Currency      string     `json:"currency"`
	ItemCount     int32      `json:"item_count"`
	SubtotalCents int64      `json:"subtotal_cents"`
	Items         []CartItem `json:"items"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// QuoteShipment models a vendor-specific shipment split during checkout.
type QuoteShipment struct {
	VendorID         string     `json:"vendor_id"`
	ItemCount        int32      `json:"item_count"`
	SubtotalCents    int64      `json:"subtotal_cents"`
	ShippingFeeCents int64      `json:"shipping_fee_cents"`
	TotalCents       int64      `json:"total_cents"`
	Items            []CartItem `json:"items"`
}

// CheckoutQuote includes order-level and shipment-level totals.
type CheckoutQuote struct {
	Currency      string          `json:"currency"`
	ItemCount     int32           `json:"item_count"`
	ShipmentCount int32           `json:"shipment_count"`
	SubtotalCents int64           `json:"subtotal_cents"`
	ShippingCents int64           `json:"shipping_cents"`
	TotalCents    int64           `json:"total_cents"`
	Shipments     []QuoteShipment `json:"shipments"`
}

// OrderShipment is the shipment representation on placed orders.
type OrderShipment struct {
	ID               string `json:"id"`
	VendorID         string `json:"vendor_id"`
	Status           string `json:"status"`
	ItemCount        int32  `json:"item_count"`
	SubtotalCents    int64  `json:"subtotal_cents"`
	ShippingFeeCents int64  `json:"shipping_fee_cents"`
	TotalCents       int64  `json:"total_cents"`
}

// OrderItem is an immutable order line snapshot.
type OrderItem struct {
	ID             string `json:"id"`
	ShipmentID     string `json:"shipment_id"`
	ProductID      string `json:"product_id"`
	VendorID       string `json:"vendor_id"`
	Title          string `json:"title"`
	Qty            int32  `json:"qty"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	LineTotalCents int64  `json:"line_total_cents"`
	Currency       string `json:"currency"`
}

// Order is created by checkout/place-order.
type Order struct {
	ID             string          `json:"id"`
	BuyerUserID    string          `json:"buyer_user_id,omitempty"`
	GuestToken     string          `json:"guest_token,omitempty"`
	Status         string          `json:"status"`
	Currency       string          `json:"currency"`
	ItemCount      int32           `json:"item_count"`
	ShipmentCount  int32           `json:"shipment_count"`
	SubtotalCents  int64           `json:"subtotal_cents"`
	ShippingCents  int64           `json:"shipping_cents"`
	DiscountCents  int64           `json:"discount_cents"`
	TaxCents       int64           `json:"tax_cents"`
	TotalCents     int64           `json:"total_cents"`
	IdempotencyKey string          `json:"idempotency_key"`
	Shipments      []OrderShipment `json:"shipments"`
	Items          []OrderItem     `json:"items"`
	CreatedAt      time.Time       `json:"created_at"`
}

type cartState struct {
	id         string
	currency   string
	items      map[string]CartItem
	byProduct  map[string]string
	orderedIDs []string
	updatedAt  time.Time
}

// Service keeps cart and checkout state in-memory for the buyer cart/checkout milestone.
type Service struct {
	mu                    sync.Mutex
	shippingFeeCents      int64
	cartsByActorKey       map[string]*cartState
	ordersByID            map[string]Order
	idempotencyToOrderKey map[string]string
}

func NewService(shippingFeeCents int64) *Service {
	fee := shippingFeeCents
	if fee <= 0 {
		fee = DefaultShippingFeeCents
	}

	return &Service{
		shippingFeeCents:      fee,
		cartsByActorKey:       make(map[string]*cartState),
		ordersByID:            make(map[string]Order),
		idempotencyToOrderKey: make(map[string]string),
	}
}

func (s *Service) GetCart(actor Actor) (Cart, error) {
	key, err := actor.key()
	if err != nil {
		return Cart{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.getOrCreateCartLocked(key)
	return snapshotCart(state), nil
}

func (s *Service) UpsertItem(actor Actor, product ProductSnapshot, qty int32) (Cart, error) {
	key, err := actor.key()
	if err != nil {
		return Cart{}, err
	}
	if qty <= 0 {
		return Cart{}, ErrInvalidQuantity
	}
	if err := validateProductSnapshot(product); err != nil {
		return Cart{}, err
	}
	if product.StockQty > 0 && qty > product.StockQty {
		return Cart{}, ErrInsufficientStock
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.getOrCreateCartLocked(key)
	if state.currency != product.Currency {
		return Cart{}, ErrCurrencyMismatch
	}

	if itemID, exists := state.byProduct[product.ID]; exists {
		line := state.items[itemID]
		line.Qty = qty
		line.AvailableStock = product.StockQty
		line.UnitPriceCents = product.UnitPriceInclTaxCents
		line.LineTotalCents = product.UnitPriceInclTaxCents * int64(qty)
		line.LastUpdatedUnix = time.Now().UTC().Unix()
		state.items[itemID] = line
		state.updatedAt = time.Now().UTC()
		return snapshotCart(state), nil
	}

	itemID := identifier.New("cit")
	line := CartItem{
		ID:              itemID,
		ProductID:       product.ID,
		VendorID:        product.VendorID,
		Title:           strings.TrimSpace(product.Title),
		Qty:             qty,
		UnitPriceCents:  product.UnitPriceInclTaxCents,
		LineTotalCents:  product.UnitPriceInclTaxCents * int64(qty),
		Currency:        product.Currency,
		AvailableStock:  product.StockQty,
		LastUpdatedUnix: time.Now().UTC().Unix(),
	}

	state.items[itemID] = line
	state.byProduct[product.ID] = itemID
	state.orderedIDs = append(state.orderedIDs, itemID)
	state.updatedAt = time.Now().UTC()

	return snapshotCart(state), nil
}

func (s *Service) UpdateItemQty(actor Actor, itemID string, qty int32) (Cart, error) {
	key, err := actor.key()
	if err != nil {
		return Cart{}, err
	}
	if qty <= 0 {
		return Cart{}, ErrInvalidQuantity
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.getOrCreateCartLocked(key)
	line, exists := state.items[itemID]
	if !exists {
		return Cart{}, ErrCartItemNotFound
	}
	if line.AvailableStock > 0 && qty > line.AvailableStock {
		return Cart{}, ErrInsufficientStock
	}

	line.Qty = qty
	line.LineTotalCents = line.UnitPriceCents * int64(qty)
	line.LastUpdatedUnix = time.Now().UTC().Unix()
	state.items[itemID] = line
	state.updatedAt = time.Now().UTC()

	return snapshotCart(state), nil
}

func (s *Service) RemoveItem(actor Actor, itemID string) (Cart, error) {
	key, err := actor.key()
	if err != nil {
		return Cart{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.getOrCreateCartLocked(key)
	line, exists := state.items[itemID]
	if !exists {
		return Cart{}, ErrCartItemNotFound
	}

	delete(state.items, itemID)
	delete(state.byProduct, line.ProductID)
	for i := range state.orderedIDs {
		if state.orderedIDs[i] == itemID {
			state.orderedIDs = append(state.orderedIDs[:i], state.orderedIDs[i+1:]...)
			break
		}
	}
	state.updatedAt = time.Now().UTC()

	return snapshotCart(state), nil
}

func (s *Service) Quote(actor Actor) (CheckoutQuote, error) {
	key, err := actor.key()
	if err != nil {
		return CheckoutQuote{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.getOrCreateCartLocked(key)
	return s.buildQuoteLocked(state)
}

func (s *Service) PlaceOrder(actor Actor, idempotencyKey string) (Order, error) {
	actorKey, err := actor.key()
	if err != nil {
		return Order{}, err
	}

	normalizedKey := strings.TrimSpace(idempotencyKey)
	if normalizedKey == "" {
		return Order{}, ErrIdempotencyKey
	}
	requestKey := actorKey + "::" + normalizedKey

	s.mu.Lock()
	defer s.mu.Unlock()

	if existingOrderID, exists := s.idempotencyToOrderKey[requestKey]; exists {
		return s.ordersByID[existingOrderID], nil
	}

	state := s.getOrCreateCartLocked(actorKey)
	quote, err := s.buildQuoteLocked(state)
	if err != nil {
		return Order{}, err
	}

	shipmentIDByVendor := make(map[string]string, len(quote.Shipments))
	shipments := make([]OrderShipment, 0, len(quote.Shipments))
	for _, shipment := range quote.Shipments {
		shipmentID := identifier.New("shp")
		shipmentIDByVendor[shipment.VendorID] = shipmentID
		shipments = append(shipments, OrderShipment{
			ID:               shipmentID,
			VendorID:         shipment.VendorID,
			Status:           ShipmentStatusPending,
			ItemCount:        shipment.ItemCount,
			SubtotalCents:    shipment.SubtotalCents,
			ShippingFeeCents: shipment.ShippingFeeCents,
			TotalCents:       shipment.TotalCents,
		})
	}

	items := make([]OrderItem, 0, len(state.orderedIDs))
	for _, itemID := range state.orderedIDs {
		line, exists := state.items[itemID]
		if !exists {
			continue
		}
		items = append(items, OrderItem{
			ID:             identifier.New("oit"),
			ShipmentID:     shipmentIDByVendor[line.VendorID],
			ProductID:      line.ProductID,
			VendorID:       line.VendorID,
			Title:          line.Title,
			Qty:            line.Qty,
			UnitPriceCents: line.UnitPriceCents,
			LineTotalCents: line.LineTotalCents,
			Currency:       line.Currency,
		})
	}

	now := time.Now().UTC()
	order := Order{
		ID:             identifier.New("ord"),
		BuyerUserID:    strings.TrimSpace(actor.BuyerUserID),
		GuestToken:     strings.TrimSpace(actor.GuestToken),
		Status:         OrderStatusPendingPayment,
		Currency:       quote.Currency,
		ItemCount:      quote.ItemCount,
		ShipmentCount:  quote.ShipmentCount,
		SubtotalCents:  quote.SubtotalCents,
		ShippingCents:  quote.ShippingCents,
		DiscountCents:  0,
		TaxCents:       0,
		TotalCents:     quote.TotalCents,
		IdempotencyKey: normalizedKey,
		Shipments:      shipments,
		Items:          items,
		CreatedAt:      now,
	}

	s.ordersByID[order.ID] = order
	s.idempotencyToOrderKey[requestKey] = order.ID

	state.items = make(map[string]CartItem)
	state.byProduct = make(map[string]string)
	state.orderedIDs = make([]string, 0)
	state.updatedAt = now

	return order, nil
}

func (s *Service) GetOrder(actor Actor, orderID string) (Order, bool, error) {
	if _, err := actor.key(); err != nil {
		return Order{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.ordersByID[orderID]
	if !exists {
		return Order{}, false, nil
	}

	if strings.TrimSpace(actor.BuyerUserID) != "" {
		if order.BuyerUserID != strings.TrimSpace(actor.BuyerUserID) {
			return Order{}, false, nil
		}
		return order, true, nil
	}

	if order.GuestToken != strings.TrimSpace(actor.GuestToken) {
		return Order{}, false, nil
	}
	return order, true, nil
}

func (s *Service) MarkOrderPaid(orderID string) (Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.ordersByID[strings.TrimSpace(orderID)]
	if !exists {
		return Order{}, false
	}
	if order.Status != OrderStatusPaid {
		order.Status = OrderStatusPaid
		s.ordersByID[order.ID] = order
	}

	return order, true
}

func (s *Service) MarkOrderPaymentFailed(orderID string) (Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.ordersByID[strings.TrimSpace(orderID)]
	if !exists {
		return Order{}, false
	}
	if order.Status == OrderStatusPaid {
		return order, true
	}
	order.Status = OrderStatusPaymentFailed
	s.ordersByID[order.ID] = order

	return order, true
}

func validateProductSnapshot(product ProductSnapshot) error {
	if strings.TrimSpace(product.ID) == "" || strings.TrimSpace(product.VendorID) == "" {
		return ErrInvalidProduct
	}
	if strings.TrimSpace(product.Title) == "" || strings.TrimSpace(product.Currency) == "" {
		return ErrInvalidProduct
	}
	if product.UnitPriceInclTaxCents < 0 {
		return ErrInvalidProduct
	}
	if product.StockQty < 0 {
		return ErrInvalidProduct
	}
	return nil
}

func (s *Service) getOrCreateCartLocked(actorKey string) *cartState {
	if existing, ok := s.cartsByActorKey[actorKey]; ok {
		return existing
	}

	now := time.Now().UTC()
	state := &cartState{
		id:         identifier.New("crt"),
		currency:   DefaultCurrency,
		items:      make(map[string]CartItem),
		byProduct:  make(map[string]string),
		orderedIDs: make([]string, 0),
		updatedAt:  now,
	}
	s.cartsByActorKey[actorKey] = state
	return state
}

func (s *Service) buildQuoteLocked(state *cartState) (CheckoutQuote, error) {
	if len(state.items) == 0 || len(state.orderedIDs) == 0 {
		return CheckoutQuote{}, ErrCartEmpty
	}

	type shipmentAccumulator struct {
		itemCount     int32
		subtotalCents int64
		items         []CartItem
	}

	byVendor := make(map[string]*shipmentAccumulator)
	vendorIDs := make([]string, 0)
	var totalItemCount int32
	var subtotal int64

	for _, itemID := range state.orderedIDs {
		line, exists := state.items[itemID]
		if !exists {
			continue
		}

		bucket, exists := byVendor[line.VendorID]
		if !exists {
			bucket = &shipmentAccumulator{items: make([]CartItem, 0, 2)}
			byVendor[line.VendorID] = bucket
			vendorIDs = append(vendorIDs, line.VendorID)
		}

		bucket.itemCount += line.Qty
		bucket.subtotalCents += line.LineTotalCents
		bucket.items = append(bucket.items, line)

		totalItemCount += line.Qty
		subtotal += line.LineTotalCents
	}

	if len(vendorIDs) == 0 {
		return CheckoutQuote{}, ErrCartEmpty
	}

	sort.Strings(vendorIDs)
	shipments := make([]QuoteShipment, 0, len(vendorIDs))
	var shippingTotal int64

	for _, vendorID := range vendorIDs {
		bucket := byVendor[vendorID]
		shipping := s.shippingFeeCents
		shipmentTotal := bucket.subtotalCents + shipping
		shipments = append(shipments, QuoteShipment{
			VendorID:         vendorID,
			ItemCount:        bucket.itemCount,
			SubtotalCents:    bucket.subtotalCents,
			ShippingFeeCents: shipping,
			TotalCents:       shipmentTotal,
			Items:            append([]CartItem(nil), bucket.items...),
		})
		shippingTotal += shipping
	}

	return CheckoutQuote{
		Currency:      state.currency,
		ItemCount:     totalItemCount,
		ShipmentCount: int32(len(shipments)),
		SubtotalCents: subtotal,
		ShippingCents: shippingTotal,
		TotalCents:    subtotal + shippingTotal,
		Shipments:     shipments,
	}, nil
}

func snapshotCart(state *cartState) Cart {
	items := make([]CartItem, 0, len(state.orderedIDs))
	var itemCount int32
	var subtotal int64

	for _, itemID := range state.orderedIDs {
		line, exists := state.items[itemID]
		if !exists {
			continue
		}
		items = append(items, line)
		itemCount += line.Qty
		subtotal += line.LineTotalCents
	}

	return Cart{
		ID:            state.id,
		Currency:      state.currency,
		ItemCount:     itemCount,
		SubtotalCents: subtotal,
		Items:         items,
		UpdatedAt:     state.updatedAt,
	}
}
