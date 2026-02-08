package router

import (
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/catalog"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/vendor"
)

func (a *api) seedDevelopmentCatalog() {
	if len(a.catalogService.ListVisibleProducts(func(string) bool { return true })) > 0 {
		return
	}

	a.catalogService.UpsertCategory("stationery", "Stationery")
	a.catalogService.UpsertCategory("prints", "Prints")
	a.catalogService.UpsertCategory("home", "Home")

	ownerA := "seed-owner-a"
	ownerB := "seed-owner-b"

	vendorA, err := a.vendorService.Register(ownerA, "north-studio", "North Studio")
	if err == nil {
		vendorA, _ = a.vendorService.SetVerificationState(vendorA.ID, vendor.VerificationVerified)
		_ = vendorA
	}
	vendorB, err := a.vendorService.Register(ownerB, "line-press", "Line Press")
	if err == nil {
		vendorB, _ = a.vendorService.SetVerificationState(vendorB.ID, vendor.VerificationVerified)
		_ = vendorB
	}

	if firstVendor, ok := a.vendorService.GetByOwner(ownerA); ok {
		a.catalogService.CreateProductWithInput(catalog.CreateProductInput{
			OwnerUserID:       ownerA,
			VendorID:          firstVendor.ID,
			Title:             "Grid Notebook",
			Description:       "A minimal notebook with soft cover and grid pages.",
			CategorySlug:      "stationery",
			Tags:              []string{"notebook", "paper", "grid"},
			PriceInclTaxCents: 2200,
			Currency:          "USD",
			StockQty:          75,
			RatingAverage:     4.8,
			Status:            catalog.ProductStatusApproved,
		})
		a.catalogService.CreateProductWithInput(catalog.CreateProductInput{
			OwnerUserID:       ownerA,
			VendorID:          firstVendor.ID,
			Title:             "Desk Weekly Planner",
			Description:       "Weekly planner with clean blocks and bold date markers.",
			CategorySlug:      "stationery",
			Tags:              []string{"planner", "desk"},
			PriceInclTaxCents: 1800,
			Currency:          "USD",
			StockQty:          41,
			RatingAverage:     4.5,
			Status:            catalog.ProductStatusApproved,
		})
	}

	if secondVendor, ok := a.vendorService.GetByOwner(ownerB); ok {
		a.catalogService.CreateProductWithInput(catalog.CreateProductInput{
			OwnerUserID:       ownerB,
			VendorID:          secondVendor.ID,
			Title:             "Monochrome Poster Print",
			Description:       "Museum-grade paper print for modern workspaces.",
			CategorySlug:      "prints",
			Tags:              []string{"poster", "print", "art"},
			PriceInclTaxCents: 4800,
			Currency:          "USD",
			StockQty:          19,
			RatingAverage:     4.9,
			Status:            catalog.ProductStatusApproved,
		})
		a.catalogService.CreateProductWithInput(catalog.CreateProductInput{
			OwnerUserID:       ownerB,
			VendorID:          secondVendor.ID,
			Title:             "Ceramic Coffee Cup",
			Description:       "Hand-glazed cup built for everyday use.",
			CategorySlug:      "home",
			Tags:              []string{"home", "ceramic", "cup"},
			PriceInclTaxCents: 3200,
			Currency:          "USD",
			StockQty:          33,
			RatingAverage:     4.4,
			Status:            catalog.ProductStatusApproved,
		})
	}
}
