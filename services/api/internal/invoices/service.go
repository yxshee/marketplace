package invoices

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/commerce"
)

var (
	ErrInvalidOrder        = errors.New("order is invalid")
	ErrOrderNotInvoiceable = errors.New("order is not invoiceable")
)

type Config struct {
	PlatformName         string
	PlatformLegalEntity  string
	PlatformSupportEmail string
	PlatformAddress      string
}

type Invoice struct {
	OrderID       string    `json:"order_id"`
	InvoiceNumber string    `json:"invoice_number"`
	FileName      string    `json:"file_name"`
	IssuedAt      time.Time `json:"issued_at"`
	Currency      string    `json:"currency"`
	TotalCents    int64     `json:"total_cents"`
	Content       []byte    `json:"-"`
}

type Service struct {
	mu        sync.Mutex
	cfg       Config
	now       func() time.Time
	sequence  int64
	byOrderID map[string]Invoice
}

func NewService(cfg Config) *Service {
	if strings.TrimSpace(cfg.PlatformName) == "" {
		cfg.PlatformName = "Marketplace Gumroad Inspired"
	}
	if strings.TrimSpace(cfg.PlatformLegalEntity) == "" {
		cfg.PlatformLegalEntity = cfg.PlatformName
	}
	if strings.TrimSpace(cfg.PlatformSupportEmail) == "" {
		cfg.PlatformSupportEmail = "support@example.com"
	}
	if strings.TrimSpace(cfg.PlatformAddress) == "" {
		cfg.PlatformAddress = "Global operations"
	}

	return &Service{
		cfg:       cfg,
		now:       func() time.Time { return time.Now().UTC() },
		sequence:  0,
		byOrderID: make(map[string]Invoice),
	}
}

func (s *Service) GenerateForOrder(order commerce.Order) (Invoice, error) {
	orderID := strings.TrimSpace(order.ID)
	if orderID == "" || order.TotalCents <= 0 || strings.TrimSpace(order.Currency) == "" || len(order.Shipments) == 0 {
		return Invoice{}, ErrInvalidOrder
	}

	switch order.Status {
	case commerce.OrderStatusPaid, commerce.OrderStatusCODConfirmed:
	default:
		return Invoice{}, ErrOrderNotInvoiceable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byOrderID[orderID]; ok {
		return existing, nil
	}

	s.sequence++
	issuedAt := s.now()
	invoiceNumber := fmt.Sprintf("INV-%s-%06d", issuedAt.Format("20060102"), s.sequence)

	content, err := renderInvoicePDF(order, invoiceNumber, issuedAt, s.cfg)
	if err != nil {
		return Invoice{}, err
	}

	invoice := Invoice{
		OrderID:       orderID,
		InvoiceNumber: invoiceNumber,
		FileName:      fmt.Sprintf("invoice-%s.pdf", strings.ToLower(invoiceNumber)),
		IssuedAt:      issuedAt,
		Currency:      order.Currency,
		TotalCents:    order.TotalCents,
		Content:       content,
	}
	s.byOrderID[orderID] = invoice

	return invoice, nil
}

func renderInvoicePDF(order commerce.Order, invoiceNumber string, issuedAt time.Time, cfg Config) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(invoiceNumber, false)
	pdf.SetAuthor(cfg.PlatformName, false)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, "Invoice", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, cfg.PlatformName, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, cfg.PlatformLegalEntity, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, cfg.PlatformSupportEmail, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, cfg.PlatformAddress, "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 6, fmt.Sprintf("Invoice number: %s", invoiceNumber), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Issued at (UTC): %s", issuedAt.Format(time.RFC3339)), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Order ID: %s", order.ID), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	for _, shipment := range order.Shipments {
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 7, fmt.Sprintf("Shipment %s", shipment.ID), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, fmt.Sprintf("Vendor: %s", shipment.VendorID), "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("Shipment status: %s", shipment.Status), "", 1, "L", false, 0, "")

		for _, item := range order.Items {
			if item.ShipmentID != shipment.ID {
				continue
			}
			pdf.CellFormat(0, 6, fmt.Sprintf("- %s x%d  (%s)", item.Title, item.Qty, formatCents(item.LineTotalCents)), "", 1, "L", false, 0, "")
		}

		pdf.CellFormat(0, 6, fmt.Sprintf("Shipment subtotal: %s", formatCents(shipment.SubtotalCents)), "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("Shipping: %s", formatCents(shipment.ShippingFeeCents)), "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("Shipment total: %s", formatCents(shipment.TotalCents)), "", 1, "L", false, 0, "")
		pdf.Ln(2)
	}

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 7, "Order totals", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Subtotal: %s", formatCents(order.SubtotalCents)), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Shipping: %s", formatCents(order.ShippingCents)), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Discount: %s", formatCents(order.DiscountCents)), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Tax included: %s", formatCents(order.TaxCents)), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Grand total: %s", formatCents(order.TotalCents)), "", 1, "L", false, 0, "")

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func formatCents(cents int64) string {
	sign := ""
	value := cents
	if cents < 0 {
		sign = "-"
		value = -cents
	}
	return fmt.Sprintf("%s$%d.%02d", sign, value/100, value%100)
}
