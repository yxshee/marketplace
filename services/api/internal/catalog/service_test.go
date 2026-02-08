package catalog

import "testing"

func TestModerationWorkflow(t *testing.T) {
	service := NewService()
	product := service.CreateProduct("usr_1", "ven_1", "Notebook", "Simple notebook", "USD", 2499)

	if _, err := service.SubmitForModeration(product.ID, "usr_2", "ven_1"); err != ErrUnauthorizedProductAccess {
		t.Fatalf("expected ErrUnauthorizedProductAccess, got %v", err)
	}

	submitted, err := service.SubmitForModeration(product.ID, "usr_1", "ven_1")
	if err != nil {
		t.Fatalf("SubmitForModeration() error = %v", err)
	}
	if submitted.Status != ProductStatusPendingApproval {
		t.Fatalf("expected pending_approval status, got %s", submitted.Status)
	}

	approved, err := service.ReviewProduct(product.ID, "admin_1", ModerationDecisionApprove, "")
	if err != nil {
		t.Fatalf("ReviewProduct() error = %v", err)
	}
	if approved.Status != ProductStatusApproved {
		t.Fatalf("expected approved status, got %s", approved.Status)
	}

	visible := service.ListVisibleProducts(func(vendorID string) bool { return vendorID == "ven_1" })
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible product, got %d", len(visible))
	}
}

func TestSearchFilteringSortingAndPagination(t *testing.T) {
	service := NewService()
	service.UpsertCategory("notebooks", "Notebooks")
	service.UpsertCategory("prints", "Prints")

	first := service.CreateProductWithInput(CreateProductInput{
		OwnerUserID:       "usr_1",
		VendorID:          "ven_1",
		Title:             "Graph Paper Notebook",
		Description:       "Dotted notebook for sketching",
		CategorySlug:      "notebooks",
		Tags:              []string{"paper", "graph"},
		PriceInclTaxCents: 1999,
		Currency:          "USD",
		RatingAverage:     4.8,
		Status:            ProductStatusApproved,
	})
	_ = first

	second := service.CreateProductWithInput(CreateProductInput{
		OwnerUserID:       "usr_2",
		VendorID:          "ven_2",
		Title:             "Poster Print",
		Description:       "A minimal line-art print",
		CategorySlug:      "prints",
		Tags:              []string{"wall", "art"},
		PriceInclTaxCents: 4999,
		Currency:          "USD",
		RatingAverage:     4.2,
		Status:            ProductStatusApproved,
	})
	_ = second

	third := service.CreateProductWithInput(CreateProductInput{
		OwnerUserID:       "usr_3",
		VendorID:          "ven_3",
		Title:             "Pocket Notebook",
		Description:       "Compact notebook",
		CategorySlug:      "notebooks",
		Tags:              []string{"paper"},
		PriceInclTaxCents: 999,
		Currency:          "USD",
		RatingAverage:     3.9,
		Status:            ProductStatusApproved,
	})
	_ = third

	result := service.Search(SearchParams{
		Query:    "notebook",
		Category: "notebooks",
		SortBy:   SortPriceAsc,
		Limit:    10,
		Offset:   0,
	}, func(vendorID string) bool { return vendorID != "ven_2" })

	if result.Total != 2 {
		t.Fatalf("expected 2 results, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 paged items, got %d", len(result.Items))
	}
	if result.Items[0].PriceInclTaxCents > result.Items[1].PriceInclTaxCents {
		t.Fatalf("expected ascending price order")
	}

	paged := service.Search(SearchParams{
		SortBy: SortNewest,
		Limit:  1,
		Offset: 1,
	}, func(vendorID string) bool { return true })
	if paged.Total != 3 {
		t.Fatalf("expected total 3, got %d", paged.Total)
	}
	if len(paged.Items) != 1 {
		t.Fatalf("expected one paged item, got %d", len(paged.Items))
	}
}

func TestVendorProductUpdateDeleteAndList(t *testing.T) {
	service := NewService()
	product := service.CreateProduct("usr_vendor", "ven_vendor", "Notebook", "Simple notebook", "USD", 2500)

	price := int64(3100)
	title := "Notebook Pro"
	updated, err := service.UpdateProduct(product.ID, "usr_vendor", "ven_vendor", UpdateProductInput{
		Title:             &title,
		PriceInclTaxCents: &price,
	})
	if err != nil {
		t.Fatalf("UpdateProduct() error = %v", err)
	}
	if updated.Title != title || updated.PriceInclTaxCents != price {
		t.Fatalf("unexpected updated product payload: %#v", updated)
	}

	if _, err := service.UpdateProduct(product.ID, "usr_other", "ven_vendor", UpdateProductInput{
		Title: &title,
	}); err != ErrUnauthorizedProductAccess {
		t.Fatalf("expected ErrUnauthorizedProductAccess, got %v", err)
	}

	items := service.ListVendorProducts("usr_vendor", "ven_vendor")
	if len(items) != 1 {
		t.Fatalf("expected one vendor product, got %d", len(items))
	}

	if err := service.DeleteProduct(product.ID, "usr_vendor", "ven_vendor"); err != nil {
		t.Fatalf("DeleteProduct() error = %v", err)
	}
	if len(service.ListVendorProducts("usr_vendor", "ven_vendor")) != 0 {
		t.Fatalf("expected no products after delete")
	}
}
