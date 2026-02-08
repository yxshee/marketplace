package router

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/commerce"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/refunds"
)

type vendorAnalyticsOverviewResponse struct {
	Currency         string                             `json:"currency"`
	RevenueCents     int64                              `json:"revenue_cents"`
	OrderCount       int                                `json:"order_count"`
	PaidOrderCount   int                                `json:"paid_order_count"`
	ShipmentCount    int                                `json:"shipment_count"`
	ConversionFunnel vendorAnalyticsConversionFunnel    `json:"conversion_funnel"`
	RefundStats      vendorAnalyticsOverviewRefundStats `json:"refund_stats"`
}

type vendorAnalyticsConversionFunnel struct {
	OrdersTotal        int `json:"orders_total"`
	OrdersPaid         int `json:"orders_paid"`
	ShipmentsTotal     int `json:"shipments_total"`
	ShipmentsShipped   int `json:"shipments_shipped"`
	ShipmentsDelivered int `json:"shipments_delivered"`
}

type vendorAnalyticsOverviewRefundStats struct {
	RequestsTotal      int `json:"requests_total"`
	PendingTotal       int `json:"pending_total"`
	ApprovedTotal      int `json:"approved_total"`
	RejectedTotal      int `json:"rejected_total"`
	ApprovalRateBPS    int `json:"approval_rate_bps"`
	OrderRefundRateBPS int `json:"order_refund_rate_bps"`
}

type vendorAnalyticsTopProduct struct {
	ProductID    string `json:"product_id"`
	Title        string `json:"title"`
	OrderCount   int    `json:"order_count"`
	UnitsSold    int32  `json:"units_sold"`
	RevenueCents int64  `json:"revenue_cents"`
}

type vendorAnalyticsTopProductsResponse struct {
	Items []vendorAnalyticsTopProduct `json:"items"`
	Total int                         `json:"total"`
}

type vendorAnalyticsCouponPerformance struct {
	CouponID               string    `json:"coupon_id"`
	Code                   string    `json:"code"`
	Active                 bool      `json:"active"`
	DiscountType           string    `json:"discount_type"`
	DiscountValue          int64     `json:"discount_value"`
	UsageCount             int       `json:"usage_count"`
	DiscountsGrantedCents  int64     `json:"discounts_granted_cents"`
	AttributedRevenueCents int64     `json:"attributed_revenue_cents"`
	ConversionRateBPS      int       `json:"conversion_rate_bps"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type vendorAnalyticsCouponsResponse struct {
	Items []vendorAnalyticsCouponPerformance `json:"items"`
	Total int                                `json:"total"`
}

func (a *api) handleVendorAnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	shipments, err := a.commerce.ListVendorShipments(registeredVendor.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to load vendor shipments")
		return
	}
	refundItems, err := a.refunds.ListVendorRequests(registeredVendor.ID, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to load vendor refunds")
		return
	}

	orderSet := make(map[string]struct{})
	paidOrderSet := make(map[string]struct{})
	revenueCents := int64(0)
	shippedCount := 0
	deliveredCount := 0
	currency := commerce.DefaultCurrency

	for _, shipment := range shipments {
		orderSet[shipment.OrderID] = struct{}{}
		currency = shipment.Currency
		if isSettledOrderStatus(shipment.OrderStatus) {
			paidOrderSet[shipment.OrderID] = struct{}{}
			if shipment.Status != commerce.ShipmentStatusCancelled {
				revenueCents += shipment.TotalCents
			}
		}
		switch shipment.Status {
		case commerce.ShipmentStatusShipped:
			shippedCount++
		case commerce.ShipmentStatusDelivered:
			shippedCount++
			deliveredCount++
		}
	}

	refundPending := 0
	refundApproved := 0
	refundRejected := 0
	for _, item := range refundItems {
		switch item.Status {
		case refunds.RequestStatusPending:
			refundPending++
		case refunds.RequestStatusApproved:
			refundApproved++
		case refunds.RequestStatusRejected:
			refundRejected++
		}
	}

	response := vendorAnalyticsOverviewResponse{
		Currency:       currency,
		RevenueCents:   revenueCents,
		OrderCount:     len(orderSet),
		PaidOrderCount: len(paidOrderSet),
		ShipmentCount:  len(shipments),
		ConversionFunnel: vendorAnalyticsConversionFunnel{
			OrdersTotal:        len(orderSet),
			OrdersPaid:         len(paidOrderSet),
			ShipmentsTotal:     len(shipments),
			ShipmentsShipped:   shippedCount,
			ShipmentsDelivered: deliveredCount,
		},
		RefundStats: vendorAnalyticsOverviewRefundStats{
			RequestsTotal:      len(refundItems),
			PendingTotal:       refundPending,
			ApprovedTotal:      refundApproved,
			RejectedTotal:      refundRejected,
			ApprovalRateBPS:    ratioBPS(refundApproved, len(refundItems)),
			OrderRefundRateBPS: ratioBPS(refundApproved, len(paidOrderSet)),
		},
	}

	writeJSON(w, http.StatusOK, response)
}

func (a *api) handleVendorAnalyticsTopProducts(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	shipments, err := a.commerce.ListVendorShipments(registeredVendor.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to load vendor shipments")
		return
	}

	type aggregate struct {
		ProductID    string
		Title        string
		UnitsSold    int32
		RevenueCents int64
		OrderSet     map[string]struct{}
	}

	aggregates := make(map[string]*aggregate)
	for _, shipment := range shipments {
		if shipment.Status == commerce.ShipmentStatusCancelled {
			continue
		}
		for _, item := range shipment.Items {
			entry, exists := aggregates[item.ProductID]
			if !exists {
				entry = &aggregate{
					ProductID: item.ProductID,
					Title:     item.Title,
					OrderSet:  make(map[string]struct{}),
				}
				aggregates[item.ProductID] = entry
			}
			entry.UnitsSold += item.Qty
			entry.RevenueCents += item.LineTotalCents
			entry.OrderSet[shipment.OrderID] = struct{}{}
		}
	}

	items := make([]vendorAnalyticsTopProduct, 0, len(aggregates))
	for _, entry := range aggregates {
		items = append(items, vendorAnalyticsTopProduct{
			ProductID:    entry.ProductID,
			Title:        entry.Title,
			OrderCount:   len(entry.OrderSet),
			UnitsSold:    entry.UnitsSold,
			RevenueCents: entry.RevenueCents,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].RevenueCents == items[j].RevenueCents {
			if items[i].UnitsSold == items[j].UnitsSold {
				return items[i].ProductID < items[j].ProductID
			}
			return items[i].UnitsSold > items[j].UnitsSold
		}
		return items[i].RevenueCents > items[j].RevenueCents
	})

	writeJSON(w, http.StatusOK, vendorAnalyticsTopProductsResponse{
		Items: items,
		Total: len(items),
	})
}

func (a *api) handleVendorAnalyticsCoupons(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	coupons := a.coupons.ListByVendor(registeredVendor.ID)
	items := make([]vendorAnalyticsCouponPerformance, 0, len(coupons))
	for _, coupon := range coupons {
		items = append(items, vendorAnalyticsCouponPerformance{
			CouponID:               coupon.ID,
			Code:                   coupon.Code,
			Active:                 coupon.Active,
			DiscountType:           string(coupon.DiscountType),
			DiscountValue:          coupon.DiscountValue,
			UsageCount:             0,
			DiscountsGrantedCents:  0,
			AttributedRevenueCents: 0,
			ConversionRateBPS:      0,
			CreatedAt:              coupon.CreatedAt,
			UpdatedAt:              coupon.UpdatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Active != items[j].Active {
			return items[i].Active
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	writeJSON(w, http.StatusOK, vendorAnalyticsCouponsResponse{
		Items: items,
		Total: len(items),
	})
}

func isSettledOrderStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case commerce.OrderStatusPaid, commerce.OrderStatusCODConfirmed:
		return true
	default:
		return false
	}
}

func ratioBPS(numerator, denominator int) int {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	return int((int64(numerator) * 10000) / int64(denominator))
}
