package catalog

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
)

type ProductStatus string

const (
	ProductStatusDraft           ProductStatus = "draft"
	ProductStatusPendingApproval ProductStatus = "pending_approval"
	ProductStatusApproved        ProductStatus = "approved"
	ProductStatusRejected        ProductStatus = "rejected"
)

type ModerationDecision string

const (
	ModerationDecisionApprove ModerationDecision = "approve"
	ModerationDecisionReject  ModerationDecision = "reject"
)

type SortOption string

const (
	SortRelevance SortOption = "relevance"
	SortNewest    SortOption = "newest"
	SortPriceAsc  SortOption = "price_low_high"
	SortPriceDesc SortOption = "price_high_low"
	SortRating    SortOption = "rating"
)

var (
	ErrProductNotFound           = errors.New("product not found")
	ErrUnauthorizedProductAccess = errors.New("unauthorized product access")
	ErrInvalidStatusTransition   = errors.New("invalid status transition")
	ErrInvalidModerationDecision = errors.New("invalid moderation decision")
	ErrInvalidProductInput       = errors.New("invalid product input")
)

// Category is a discoverable category in the buyer catalog.
type Category struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Product models the core catalog aggregate used in foundation branches.
type Product struct {
	ID                string        `json:"id"`
	VendorID          string        `json:"vendor_id"`
	OwnerUserID       string        `json:"owner_user_id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	CategorySlug      string        `json:"category_slug"`
	Tags              []string      `json:"tags"`
	PriceInclTaxCents int64         `json:"price_incl_tax_cents"`
	Currency          string        `json:"currency"`
	StockQty          int32         `json:"stock_qty"`
	RatingAverage     float64       `json:"rating_average"`
	Status            ProductStatus `json:"status"`
	ModerationReason  string        `json:"moderation_reason,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type CreateProductInput struct {
	OwnerUserID       string
	VendorID          string
	Title             string
	Description       string
	CategorySlug      string
	Tags              []string
	PriceInclTaxCents int64
	Currency          string
	StockQty          int32
	RatingAverage     float64
	Status            ProductStatus
}

type SearchParams struct {
	Query     string
	Category  string
	VendorID  string
	PriceMin  int64
	PriceMax  int64
	MinRating float64
	SortBy    SortOption
	Limit     int
	Offset    int
}

type SearchResult struct {
	Items []Product
	Total int
}

type UpdateProductInput struct {
	Title             *string
	Description       *string
	CategorySlug      *string
	Tags              *[]string
	PriceInclTaxCents *int64
	Currency          *string
	StockQty          *int32
}

// Service provides product and moderation workflow operations.
type Service struct {
	mu            sync.RWMutex
	byID          map[string]Product
	ordered       []string
	categories    map[string]Category
	categoryOrder []string
}

func NewService() *Service {
	service := &Service{
		byID:       make(map[string]Product),
		categories: make(map[string]Category),
	}
	service.UpsertCategory("general", "General")
	return service
}

func (s *Service) UpsertCategory(slug, name string) {
	normalizedSlug := strings.ToLower(strings.TrimSpace(slug))
	normalizedName := strings.TrimSpace(name)
	if normalizedSlug == "" || normalizedName == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.categories[normalizedSlug]; !exists {
		s.categoryOrder = append(s.categoryOrder, normalizedSlug)
	}
	s.categories[normalizedSlug] = Category{Slug: normalizedSlug, Name: normalizedName}
}

func (s *Service) ListCategories() []Category {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := make([]Category, 0, len(s.categoryOrder))
	for _, slug := range s.categoryOrder {
		categories = append(categories, s.categories[slug])
	}
	return categories
}

func (s *Service) CreateProduct(ownerUserID, vendorID, title, description, currency string, priceInclTaxCents int64) Product {
	return s.CreateProductWithInput(CreateProductInput{
		OwnerUserID:       ownerUserID,
		VendorID:          vendorID,
		Title:             title,
		Description:       description,
		CategorySlug:      "general",
		Tags:              []string{},
		PriceInclTaxCents: priceInclTaxCents,
		Currency:          currency,
		StockQty:          0,
		RatingAverage:     0,
		Status:            ProductStatusDraft,
	})
}

func (s *Service) CreateProductWithInput(input CreateProductInput) Product {
	now := time.Now().UTC()
	category := strings.ToLower(strings.TrimSpace(input.CategorySlug))
	if category == "" {
		category = "general"
	}

	product := Product{
		ID:                identifier.New("prd"),
		OwnerUserID:       input.OwnerUserID,
		VendorID:          input.VendorID,
		Title:             strings.TrimSpace(input.Title),
		Description:       strings.TrimSpace(input.Description),
		CategorySlug:      category,
		Tags:              normalizeTags(input.Tags),
		PriceInclTaxCents: input.PriceInclTaxCents,
		Currency:          strings.ToUpper(strings.TrimSpace(input.Currency)),
		StockQty:          input.StockQty,
		RatingAverage:     input.RatingAverage,
		Status:            input.Status,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if product.Status == "" {
		product.Status = ProductStatusDraft
	}

	s.UpsertCategory(category, categoryDisplayName(category))

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[product.ID] = product
	s.ordered = append(s.ordered, product.ID)
	return product
}

func (s *Service) GetProductByID(productID string) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	product, exists := s.byID[productID]
	return product, exists
}

func (s *Service) ListVendorProducts(ownerUserID, vendorID string) []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Product, 0)
	for i := len(s.ordered) - 1; i >= 0; i-- {
		product := s.byID[s.ordered[i]]
		if product.OwnerUserID == ownerUserID && product.VendorID == vendorID {
			items = append(items, product)
		}
	}
	return items
}

func (s *Service) UpdateProduct(productID, ownerUserID, vendorID string, input UpdateProductInput) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.byID[productID]
	if !exists {
		return Product{}, ErrProductNotFound
	}
	if product.OwnerUserID != ownerUserID || product.VendorID != vendorID {
		return Product{}, ErrUnauthorizedProductAccess
	}

	contentChanged := false

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return Product{}, ErrInvalidProductInput
		}
		if title != product.Title {
			product.Title = title
			contentChanged = true
		}
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if description != product.Description {
			product.Description = description
			contentChanged = true
		}
	}
	if input.CategorySlug != nil {
		category := strings.ToLower(strings.TrimSpace(*input.CategorySlug))
		if category == "" {
			return Product{}, ErrInvalidProductInput
		}
		if category != product.CategorySlug {
			product.CategorySlug = category
			s.categories[category] = Category{Slug: category, Name: categoryDisplayName(category)}
			if !containsString(s.categoryOrder, category) {
				s.categoryOrder = append(s.categoryOrder, category)
			}
			contentChanged = true
		}
	}
	if input.Tags != nil {
		nextTags := normalizeTags(*input.Tags)
		product.Tags = nextTags
		contentChanged = true
	}
	if input.PriceInclTaxCents != nil {
		if *input.PriceInclTaxCents <= 0 {
			return Product{}, ErrInvalidProductInput
		}
		if *input.PriceInclTaxCents != product.PriceInclTaxCents {
			product.PriceInclTaxCents = *input.PriceInclTaxCents
			contentChanged = true
		}
	}
	if input.Currency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*input.Currency))
		if currency == "" {
			return Product{}, ErrInvalidProductInput
		}
		if currency != product.Currency {
			product.Currency = currency
			contentChanged = true
		}
	}
	if input.StockQty != nil {
		if *input.StockQty < 0 {
			return Product{}, ErrInvalidProductInput
		}
		product.StockQty = *input.StockQty
	}

	if contentChanged && product.Status == ProductStatusApproved {
		product.Status = ProductStatusDraft
		product.ModerationReason = ""
	}
	product.UpdatedAt = time.Now().UTC()
	s.byID[productID] = product

	return product, nil
}

func (s *Service) DeleteProduct(productID, ownerUserID, vendorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.byID[productID]
	if !exists {
		return ErrProductNotFound
	}
	if product.OwnerUserID != ownerUserID || product.VendorID != vendorID {
		return ErrUnauthorizedProductAccess
	}

	delete(s.byID, productID)
	filtered := s.ordered[:0]
	for _, id := range s.ordered {
		if id != productID {
			filtered = append(filtered, id)
		}
	}
	s.ordered = filtered
	return nil
}

func (s *Service) SubmitForModeration(productID, ownerUserID, vendorID string) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.byID[productID]
	if !exists {
		return Product{}, ErrProductNotFound
	}
	if product.OwnerUserID != ownerUserID || product.VendorID != vendorID {
		return Product{}, ErrUnauthorizedProductAccess
	}
	if product.Status != ProductStatusDraft && product.Status != ProductStatusRejected {
		return Product{}, ErrInvalidStatusTransition
	}

	product.Status = ProductStatusPendingApproval
	product.ModerationReason = ""
	product.UpdatedAt = time.Now().UTC()
	s.byID[productID] = product
	return product, nil
}

func (s *Service) ReviewProduct(productID, reviewerID string, decision ModerationDecision, reason string) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.byID[productID]
	if !exists {
		return Product{}, ErrProductNotFound
	}
	if product.Status != ProductStatusPendingApproval {
		return Product{}, ErrInvalidStatusTransition
	}
	if reviewerID == "" {
		return Product{}, ErrUnauthorizedProductAccess
	}

	switch decision {
	case ModerationDecisionApprove:
		product.Status = ProductStatusApproved
		product.ModerationReason = ""
	case ModerationDecisionReject:
		product.Status = ProductStatusRejected
		product.ModerationReason = strings.TrimSpace(reason)
	default:
		return Product{}, ErrInvalidModerationDecision
	}

	product.UpdatedAt = time.Now().UTC()
	s.byID[productID] = product
	return product, nil
}

func (s *Service) ListVisibleProducts(vendorVisible func(vendorID string) bool) []Product {
	result := s.Search(SearchParams{SortBy: SortNewest, Limit: 100, Offset: 0}, vendorVisible)
	return result.Items
}

func (s *Service) ListByStatus(status ProductStatus) []Product {
	target := ProductStatus(strings.TrimSpace(string(status)))
	if target == "" {
		return []Product{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Product, 0, len(s.byID))
	for _, product := range s.byID {
		if product.Status != target {
			continue
		}
		items = append(items, product)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	return items
}

func (s *Service) Search(params SearchParams, vendorVisible func(vendorID string) bool) SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := strings.ToLower(strings.TrimSpace(params.Query))
	category := strings.ToLower(strings.TrimSpace(params.Category))
	vendorID := strings.TrimSpace(params.VendorID)
	limit := params.Limit
	offset := params.Offset
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	type scoredProduct struct {
		product Product
		score   int
	}

	matches := make([]scoredProduct, 0)
	for _, productID := range s.ordered {
		product := s.byID[productID]
		if product.Status != ProductStatusApproved {
			continue
		}
		if vendorVisible != nil && !vendorVisible(product.VendorID) {
			continue
		}
		if category != "" && product.CategorySlug != category {
			continue
		}
		if vendorID != "" && product.VendorID != vendorID {
			continue
		}
		if params.PriceMin > 0 && product.PriceInclTaxCents < params.PriceMin {
			continue
		}
		if params.PriceMax > 0 && product.PriceInclTaxCents > params.PriceMax {
			continue
		}
		if params.MinRating > 0 && product.RatingAverage < params.MinRating {
			continue
		}

		score := relevanceScore(product, query)
		if query != "" && score == 0 {
			continue
		}

		matches = append(matches, scoredProduct{product: product, score: score})
	}

	sortBy := params.SortBy
	if sortBy == "" {
		if query != "" {
			sortBy = SortRelevance
		} else {
			sortBy = SortNewest
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		left := matches[i]
		right := matches[j]

		switch sortBy {
		case SortPriceAsc:
			if left.product.PriceInclTaxCents == right.product.PriceInclTaxCents {
				return left.product.ID < right.product.ID
			}
			return left.product.PriceInclTaxCents < right.product.PriceInclTaxCents
		case SortPriceDesc:
			if left.product.PriceInclTaxCents == right.product.PriceInclTaxCents {
				return left.product.ID < right.product.ID
			}
			return left.product.PriceInclTaxCents > right.product.PriceInclTaxCents
		case SortRating:
			if left.product.RatingAverage == right.product.RatingAverage {
				return left.product.ID < right.product.ID
			}
			return left.product.RatingAverage > right.product.RatingAverage
		case SortNewest:
			if left.product.CreatedAt.Equal(right.product.CreatedAt) {
				return left.product.ID < right.product.ID
			}
			return left.product.CreatedAt.After(right.product.CreatedAt)
		default:
			if left.score == right.score {
				if left.product.CreatedAt.Equal(right.product.CreatedAt) {
					return left.product.ID < right.product.ID
				}
				return left.product.CreatedAt.After(right.product.CreatedAt)
			}
			return left.score > right.score
		}
	})

	total := len(matches)
	if offset >= total {
		return SearchResult{Items: []Product{}, Total: total}
	}

	end := offset + limit
	if end > total {
		end = total
	}

	items := make([]Product, 0, end-offset)
	for _, value := range matches[offset:end] {
		items = append(items, value.product)
	}

	return SearchResult{Items: items, Total: total}
}

func relevanceScore(product Product, query string) int {
	if query == "" {
		return 1
	}

	score := 0
	lowerTitle := strings.ToLower(product.Title)
	lowerDescription := strings.ToLower(product.Description)

	if strings.Contains(lowerTitle, query) {
		score += 3
	}
	if strings.Contains(lowerDescription, query) {
		score += 1
	}
	for _, tag := range product.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			score += 2
		}
	}

	return score
}

func categoryDisplayName(slug string) string {
	parts := strings.Split(strings.ReplaceAll(slug, "-", " "), " ")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func normalizeTags(raw []string) []string {
	tags := make([]string, 0, len(raw))
	seen := make(map[string]struct{})
	for _, value := range raw {
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		tags = append(tags, trimmed)
	}
	return tags
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
