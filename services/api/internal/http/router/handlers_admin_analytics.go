package router

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/catalog"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/commerce"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/refunds"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/vendors"
)

type adminDashboardOverviewResponse struct {
	Currency              string                               `json:"currency"`
	PlatformRevenueCents  int64                                `json:"platform_revenue_cents"`
	CommissionEarnedCents int64                                `json:"commission_earned_cents"`
	OrderVolumes          adminDashboardOrderVolumes           `json:"order_volumes"`
	VendorMetrics         adminDashboardVendorMetrics          `json:"vendor_metrics"`
	ModerationQueue       adminDashboardModerationQueueMetrics `json:"moderation_queue"`
	Disputes              adminDashboardDisputeMetrics         `json:"disputes"`
	GeneratedAt           time.Time                            `json:"generated_at"`
}

type adminDashboardOrderVolumes struct {
	Total          int `json:"total"`
	PendingPayment int `json:"pending_payment"`
	CODConfirmed   int `json:"cod_confirmed"`
	Paid           int `json:"paid"`
	PaymentFailed  int `json:"payment_failed"`
}

type adminDashboardVendorMetrics struct {
	TotalVendors          int `json:"total_vendors"`
	PendingVerification   int `json:"pending_verification"`
	Verified              int `json:"verified"`
	Rejected              int `json:"rejected"`
	Suspended             int `json:"suspended"`
	ActiveWithSales       int `json:"active_with_sales"`
	VendorsWithRefundRisk int `json:"vendors_with_refund_risk"`
}

type adminDashboardModerationQueueMetrics struct {
	PendingProducts int `json:"pending_products"`
}

type adminDashboardDisputeMetrics struct {
	RefundRequestsTotal int `json:"refund_requests_total"`
	PendingTotal        int `json:"pending_total"`
	ApprovedTotal       int `json:"approved_total"`
	RejectedTotal       int `json:"rejected_total"`
}

type adminAnalyticsRevenueSummary struct {
	SettledOrdersTotal     int   `json:"settled_orders_total"`
	GrossRevenueCents      int64 `json:"gross_revenue_cents"`
	CommissionEarnedCents  int64 `json:"commission_earned_cents"`
	AverageOrderValueCents int64 `json:"average_order_value_cents"`
}

type adminAnalyticsRevenuePoint struct {
	Date                  string `json:"date"`
	OrderCount            int    `json:"order_count"`
	GrossRevenueCents     int64  `json:"gross_revenue_cents"`
	CommissionEarnedCents int64  `json:"commission_earned_cents"`
}

type adminAnalyticsRevenueResponse struct {
	Currency   string                       `json:"currency"`
	WindowDays int                          `json:"window_days"`
	Summary    adminAnalyticsRevenueSummary `json:"summary"`
	Points     []adminAnalyticsRevenuePoint `json:"points"`
}

type adminVendorAnalyticsItem struct {
	VendorID                  string                    `json:"vendor_id"`
	Slug                      string                    `json:"slug"`
	DisplayName               string                    `json:"display_name"`
	VerificationState         vendors.VerificationState `json:"verification_state"`
	CommissionBPS             int32                     `json:"commission_bps"`
	OrderCount                int                       `json:"order_count"`
	SettledOrderCount         int                       `json:"settled_order_count"`
	GrossRevenueCents         int64                     `json:"gross_revenue_cents"`
	CommissionEarnedCents     int64                     `json:"commission_earned_cents"`
	ShipmentCount             int                       `json:"shipment_count"`
	PendingShipmentCount      int                       `json:"pending_shipment_count"`
	ShippedShipmentCount      int                       `json:"shipped_shipment_count"`
	DeliveredShipmentCount    int                       `json:"delivered_shipment_count"`
	CancelledShipmentCount    int                       `json:"cancelled_shipment_count"`
	RefundRequestsTotal       int                       `json:"refund_requests_total"`
	RefundPendingTotal        int                       `json:"refund_pending_total"`
	RefundApprovedTotal       int                       `json:"refund_approved_total"`
	RefundRejectedTotal       int                       `json:"refund_rejected_total"`
	RefundApprovalRateBPS     int                       `json:"refund_approval_rate_bps"`
	SettledOrderRefundRateBPS int                       `json:"settled_order_refund_rate_bps"`
}

type adminAnalyticsVendorsResponse struct {
	Items []adminVendorAnalyticsItem `json:"items"`
	Total int                        `json:"total"`
}

type adminVendorPerformanceStats struct {
	Item adminVendorAnalyticsItem
}

type adminRevenueAccumulator struct {
	OrderCount            int
	GrossRevenueCents     int64
	CommissionEarnedCents int64
}

func (a *api) handleAdminDashboardOverview(w http.ResponseWriter, r *http.Request) {
	orders, err := a.commerce.ListOrders("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load orders")
		return
	}

	vendorList := a.vendorService.List(nil)
	vendorPerformance, err := a.buildAdminVendorPerformance(vendorList)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load vendor analytics")
		return
	}

	moderationQueue := a.catalogService.ListByStatus(catalog.ProductStatusPendingApproval)

	orderVolumes := adminDashboardOrderVolumes{
		Total: len(orders),
	}
	currency := commerce.DefaultCurrency
	platformRevenueCents := int64(0)
	commissionEarnedCents := int64(0)
	for _, order := range orders {
		switch strings.ToLower(strings.TrimSpace(order.Status)) {
		case commerce.OrderStatusPendingPayment:
			orderVolumes.PendingPayment++
		case commerce.OrderStatusCODConfirmed:
			orderVolumes.CODConfirmed++
		case commerce.OrderStatusPaid:
			orderVolumes.Paid++
		case commerce.OrderStatusPaymentFailed:
			orderVolumes.PaymentFailed++
		}
		if normalized := strings.TrimSpace(order.Currency); normalized != "" {
			currency = normalized
		}
		gross, commission := a.settledOrderFinancials(order, vendorList)
		platformRevenueCents += gross
		commissionEarnedCents += commission
	}

	disputes := adminDashboardDisputeMetrics{}
	vendorMetrics := adminDashboardVendorMetrics{
		TotalVendors: len(vendorList),
	}
	for _, vendor := range vendorList {
		switch vendor.VerificationState {
		case vendors.VerificationPending:
			vendorMetrics.PendingVerification++
		case vendors.VerificationVerified:
			vendorMetrics.Verified++
		case vendors.VerificationRejected:
			vendorMetrics.Rejected++
		case vendors.VerificationSuspended:
			vendorMetrics.Suspended++
		}
	}

	for _, performance := range vendorPerformance {
		disputes.RefundRequestsTotal += performance.Item.RefundRequestsTotal
		disputes.PendingTotal += performance.Item.RefundPendingTotal
		disputes.ApprovedTotal += performance.Item.RefundApprovedTotal
		disputes.RejectedTotal += performance.Item.RefundRejectedTotal
		if performance.Item.SettledOrderCount > 0 {
			vendorMetrics.ActiveWithSales++
		}
		if performance.Item.SettledOrderRefundRateBPS > 0 {
			vendorMetrics.VendorsWithRefundRisk++
		}
	}

	writeJSON(w, http.StatusOK, adminDashboardOverviewResponse{
		Currency:              currency,
		PlatformRevenueCents:  platformRevenueCents,
		CommissionEarnedCents: commissionEarnedCents,
		OrderVolumes:          orderVolumes,
		VendorMetrics:         vendorMetrics,
		ModerationQueue: adminDashboardModerationQueueMetrics{
			PendingProducts: len(moderationQueue),
		},
		Disputes:    disputes,
		GeneratedAt: time.Now().UTC(),
	})
}

func (a *api) handleAdminAnalyticsRevenue(w http.ResponseWriter, r *http.Request) {
	windowDays := 30
	if rawDays := strings.TrimSpace(r.URL.Query().Get("days")); rawDays != "" {
		parsedDays, err := strconv.Atoi(rawDays)
		if err != nil || parsedDays < 1 || parsedDays > 365 {
			writeError(w, http.StatusBadRequest, "days must be between 1 and 365")
			return
		}
		windowDays = parsedDays
	}

	orders, err := a.commerce.ListOrders("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load orders")
		return
	}
	vendorList := a.vendorService.List(nil)

	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, -(windowDays - 1))

	buckets := make(map[string]adminRevenueAccumulator, windowDays)
	summary := adminAnalyticsRevenueSummary{}
	currency := commerce.DefaultCurrency

	for _, order := range orders {
		if !isSettledOrderStatus(order.Status) {
			continue
		}
		createdAt := order.CreatedAt.UTC()
		if createdAt.Before(start) {
			continue
		}

		dateKey := createdAt.Format("2006-01-02")
		gross, commission := a.settledOrderFinancials(order, vendorList)
		if normalized := strings.TrimSpace(order.Currency); normalized != "" {
			currency = normalized
		}

		bucket := buckets[dateKey]
		bucket.OrderCount++
		bucket.GrossRevenueCents += gross
		bucket.CommissionEarnedCents += commission
		buckets[dateKey] = bucket

		summary.SettledOrdersTotal++
		summary.GrossRevenueCents += gross
		summary.CommissionEarnedCents += commission
	}

	if summary.SettledOrdersTotal > 0 {
		summary.AverageOrderValueCents = summary.GrossRevenueCents / int64(summary.SettledOrdersTotal)
	}

	points := make([]adminAnalyticsRevenuePoint, 0, windowDays)
	for i := 0; i < windowDays; i++ {
		day := start.AddDate(0, 0, i)
		dayKey := day.Format("2006-01-02")
		bucket := buckets[dayKey]
		points = append(points, adminAnalyticsRevenuePoint{
			Date:                  dayKey,
			OrderCount:            bucket.OrderCount,
			GrossRevenueCents:     bucket.GrossRevenueCents,
			CommissionEarnedCents: bucket.CommissionEarnedCents,
		})
	}

	writeJSON(w, http.StatusOK, adminAnalyticsRevenueResponse{
		Currency:   currency,
		WindowDays: windowDays,
		Summary:    summary,
		Points:     points,
	})
}

func (a *api) handleAdminAnalyticsVendors(w http.ResponseWriter, r *http.Request) {
	vendorList := a.vendorService.List(nil)
	performance, err := a.buildAdminVendorPerformance(vendorList)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load vendor analytics")
		return
	}

	items := make([]adminVendorAnalyticsItem, 0, len(performance))
	for _, stats := range performance {
		items = append(items, stats.Item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].GrossRevenueCents == items[j].GrossRevenueCents {
			if items[i].SettledOrderCount == items[j].SettledOrderCount {
				return items[i].VendorID < items[j].VendorID
			}
			return items[i].SettledOrderCount > items[j].SettledOrderCount
		}
		return items[i].GrossRevenueCents > items[j].GrossRevenueCents
	})

	writeJSON(w, http.StatusOK, adminAnalyticsVendorsResponse{
		Items: items,
		Total: len(items),
	})
}

func (a *api) buildAdminVendorPerformance(vendorList []vendors.Vendor) ([]adminVendorPerformanceStats, error) {
	performance := make([]adminVendorPerformanceStats, 0, len(vendorList))

	for _, vendor := range vendorList {
		shipments, err := a.commerce.ListVendorShipments(vendor.ID)
		if err != nil {
			return nil, err
		}
		refundItems, err := a.refunds.ListVendorRequests(vendor.ID, "")
		if err != nil {
			return nil, err
		}

		commissionBPS := a.defaultCommBPS
		if vendor.CommissionOverrideBPS != nil {
			commissionBPS = *vendor.CommissionOverrideBPS
		}

		orderSet := make(map[string]struct{})
		settledOrderSet := make(map[string]struct{})
		grossRevenueCents := int64(0)
		commissionEarnedCents := int64(0)

		pendingShipmentCount := 0
		shippedShipmentCount := 0
		deliveredShipmentCount := 0
		cancelledShipmentCount := 0
		for _, shipment := range shipments {
			orderSet[shipment.OrderID] = struct{}{}
			if isSettledOrderStatus(shipment.OrderStatus) {
				settledOrderSet[shipment.OrderID] = struct{}{}
				if shipment.Status != commerce.ShipmentStatusCancelled {
					grossRevenueCents += shipment.TotalCents
					commissionEarnedCents += (shipment.TotalCents * int64(commissionBPS)) / 10000
				}
			}
			switch shipment.Status {
			case commerce.ShipmentStatusPending:
				pendingShipmentCount++
			case commerce.ShipmentStatusPacked, commerce.ShipmentStatusShipped:
				shippedShipmentCount++
			case commerce.ShipmentStatusDelivered:
				deliveredShipmentCount++
			case commerce.ShipmentStatusCancelled:
				cancelledShipmentCount++
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

		performance = append(performance, adminVendorPerformanceStats{
			Item: adminVendorAnalyticsItem{
				VendorID:                  vendor.ID,
				Slug:                      vendor.Slug,
				DisplayName:               vendor.DisplayName,
				VerificationState:         vendor.VerificationState,
				CommissionBPS:             commissionBPS,
				OrderCount:                len(orderSet),
				SettledOrderCount:         len(settledOrderSet),
				GrossRevenueCents:         grossRevenueCents,
				CommissionEarnedCents:     commissionEarnedCents,
				ShipmentCount:             len(shipments),
				PendingShipmentCount:      pendingShipmentCount,
				ShippedShipmentCount:      shippedShipmentCount,
				DeliveredShipmentCount:    deliveredShipmentCount,
				CancelledShipmentCount:    cancelledShipmentCount,
				RefundRequestsTotal:       len(refundItems),
				RefundPendingTotal:        refundPending,
				RefundApprovedTotal:       refundApproved,
				RefundRejectedTotal:       refundRejected,
				RefundApprovalRateBPS:     ratioBPS(refundApproved, len(refundItems)),
				SettledOrderRefundRateBPS: ratioBPS(refundApproved, len(settledOrderSet)),
			},
		})
	}

	return performance, nil
}

func (a *api) settledOrderFinancials(order commerce.Order, vendorList []vendors.Vendor) (int64, int64) {
	if !isSettledOrderStatus(order.Status) {
		return 0, 0
	}

	commissionByVendor := make(map[string]int32, len(vendorList))
	for _, vendor := range vendorList {
		commissionBPS := a.defaultCommBPS
		if vendor.CommissionOverrideBPS != nil {
			commissionBPS = *vendor.CommissionOverrideBPS
		}
		commissionByVendor[vendor.ID] = commissionBPS
	}

	grossRevenueCents := int64(0)
	commissionEarnedCents := int64(0)
	for _, shipment := range order.Shipments {
		if shipment.Status == commerce.ShipmentStatusCancelled {
			continue
		}
		grossRevenueCents += shipment.TotalCents
		commissionBPS, exists := commissionByVendor[shipment.VendorID]
		if !exists {
			commissionBPS = a.defaultCommBPS
		}
		commissionEarnedCents += (shipment.TotalCents * int64(commissionBPS)) / 10000
	}

	return grossRevenueCents, commissionEarnedCents
}
