package router

import (
	"github.com/yxshee/marketplace-platform/services/api/internal/catalog"
	"github.com/yxshee/marketplace-platform/services/api/internal/vendors"
)

func (a *api) seedDevelopmentCatalog() {
	if len(a.catalogService.ListVisibleProducts(func(string) bool { return true })) > 0 {
		return
	}

	for _, c := range []struct {
		slug string
		name string
	}{
		{slug: "stationery", name: "Stationery"},
		{slug: "prints", name: "Prints"},
		{slug: "home", name: "Home"},
		{slug: "desk", name: "Desk"},
		{slug: "kitchen", name: "Kitchen"},
		{slug: "accessories", name: "Accessories"},
		{slug: "apparel", name: "Apparel"},
		{slug: "outdoors", name: "Outdoors"},
	} {
		a.catalogService.UpsertCategory(c.slug, c.name)
	}

	type seedVendor struct {
		ownerUserID string
		slug        string
		displayName string
	}

	vendorsToSeed := []seedVendor{
		{ownerUserID: "seed-owner-a", slug: "north-studio", displayName: "North Studio"},
		{ownerUserID: "seed-owner-b", slug: "line-press", displayName: "Line Press"},
		{ownerUserID: "seed-owner-c", slug: "koi-workshop", displayName: "Koi Workshop"},
		{ownerUserID: "seed-owner-d", slug: "sunbeam-supply", displayName: "Sunbeam Supply"},
		{ownerUserID: "seed-owner-e", slug: "fern-atelier", displayName: "Fern Atelier"},
		{ownerUserID: "seed-owner-f", slug: "moss-and-mortar", displayName: "Moss & Mortar"},
	}

	vendorIDByOwner := make(map[string]string, len(vendorsToSeed))
	for _, v := range vendorsToSeed {
		_, _ = a.vendorService.Register(v.ownerUserID, v.slug, v.displayName)

		vendor, ok := a.vendorService.GetByOwner(v.ownerUserID)
		if !ok {
			continue
		}

		_, _ = a.vendorService.SetVerificationState(vendor.ID, vendors.VerificationVerified)
		vendorIDByOwner[v.ownerUserID] = vendor.ID
	}

	type seedProduct struct {
		ownerUserID       string
		title             string
		description       string
		categorySlug      string
		tags              []string
		priceInclTaxCents int64
		stockQty          int32
		ratingAverage     float64
	}

	productsToSeed := []seedProduct{
		// North Studio
		{
			ownerUserID:       "seed-owner-a",
			title:             "Grid Notebook",
			description:       "A minimal notebook with soft cover and grid pages.",
			categorySlug:      "stationery",
			tags:              []string{"notebook", "paper", "grid"},
			priceInclTaxCents: 2200,
			stockQty:          75,
			ratingAverage:     4.8,
		},
		{
			ownerUserID:       "seed-owner-a",
			title:             "Desk Weekly Planner",
			description:       "Weekly planner with clean blocks and bold date markers.",
			categorySlug:      "stationery",
			tags:              []string{"planner", "desk"},
			priceInclTaxCents: 1800,
			stockQty:          41,
			ratingAverage:     4.5,
		},
		{
			ownerUserID:       "seed-owner-a",
			title:             "Dot-Grid Pocket Journal",
			description:       "Pocket journal with dot grid pages and a sturdy soft cover.",
			categorySlug:      "stationery",
			tags:              []string{"journal", "dot-grid", "pocket"},
			priceInclTaxCents: 2400,
			stockQty:          60,
			ratingAverage:     4.6,
		},
		{
			ownerUserID:       "seed-owner-a",
			title:             "Mechanical Pencil Set (2 pack)",
			description:       "Two precise mechanical pencils with a clean matte finish.",
			categorySlug:      "desk",
			tags:              []string{"pencil", "writing", "desk"},
			priceInclTaxCents: 1600,
			stockQty:          120,
			ratingAverage:     4.4,
		},
		{
			ownerUserID:       "seed-owner-a",
			title:             "Sticker Sheet: Minimal Icons",
			description:       "A small sheet of crisp icon stickers for planners and notes.",
			categorySlug:      "stationery",
			tags:              []string{"stickers", "planner"},
			priceInclTaxCents: 900,
			stockQty:          220,
			ratingAverage:     4.3,
		},

		// Line Press
		{
			ownerUserID:       "seed-owner-b",
			title:             "Monochrome Poster Print",
			description:       "Museum-grade paper print for modern workspaces.",
			categorySlug:      "prints",
			tags:              []string{"poster", "print", "art"},
			priceInclTaxCents: 4800,
			stockQty:          19,
			ratingAverage:     4.9,
		},
		{
			ownerUserID:       "seed-owner-b",
			title:             "Ceramic Coffee Cup",
			description:       "Hand-glazed cup built for everyday use.",
			categorySlug:      "home",
			tags:              []string{"home", "ceramic", "cup"},
			priceInclTaxCents: 3200,
			stockQty:          33,
			ratingAverage:     4.4,
		},
		{
			ownerUserID:       "seed-owner-b",
			title:             "Risograph Print: Blue Hour",
			description:       "Two-color risograph print with soft ink texture and clean lines.",
			categorySlug:      "prints",
			tags:              []string{"risograph", "print", "art"},
			priceInclTaxCents: 5200,
			stockQty:          14,
			ratingAverage:     4.8,
		},
		{
			ownerUserID:       "seed-owner-b",
			title:             "Desk Calendar (12 months)",
			description:       "A desk calendar with generous whitespace and bold dates.",
			categorySlug:      "stationery",
			tags:              []string{"calendar", "desk"},
			priceInclTaxCents: 2600,
			stockQty:          42,
			ratingAverage:     4.6,
		},
		{
			ownerUserID:       "seed-owner-b",
			title:             "Tea Towel: Graph Check",
			description:       "Woven cotton towel with a subtle grid pattern.",
			categorySlug:      "home",
			tags:              []string{"kitchen", "towel", "home"},
			priceInclTaxCents: 2000,
			stockQty:          70,
			ratingAverage:     4.5,
		},

		// Koi Workshop
		{
			ownerUserID:       "seed-owner-c",
			title:             "Aluminum Cable Clips",
			description:       "Keep charging cables and cords anchored without adhesive mess.",
			categorySlug:      "desk",
			tags:              []string{"cables", "organize", "desk"},
			priceInclTaxCents: 1500,
			stockQty:          110,
			ratingAverage:     4.5,
		},
		{
			ownerUserID:       "seed-owner-c",
			title:             "Hardwood Desk Tray",
			description:       "A slim catch-all tray for keys, pens, and daily carry.",
			categorySlug:      "desk",
			tags:              []string{"tray", "wood", "desk"},
			priceInclTaxCents: 5400,
			stockQty:          18,
			ratingAverage:     4.9,
		},
		{
			ownerUserID:       "seed-owner-c",
			title:             "Enamel Pin (Sunset)",
			description:       "Small enamel pin with a bright color pop for bags and jackets.",
			categorySlug:      "accessories",
			tags:              []string{"pin", "enamel"},
			priceInclTaxCents: 1200,
			stockQty:          200,
			ratingAverage:     4.3,
		},
		{
			ownerUserID:       "seed-owner-c",
			title:             "Canvas Tote Bag",
			description:       "Midweight tote with reinforced handles and interior pocket.",
			categorySlug:      "accessories",
			tags:              []string{"tote", "canvas", "bag"},
			priceInclTaxCents: 3600,
			stockQty:          44,
			ratingAverage:     4.6,
		},
		{
			ownerUserID:       "seed-owner-c",
			title:             "Keyring Carabiner",
			description:       "Compact carabiner keyring for easy carry and quick access.",
			categorySlug:      "accessories",
			tags:              []string{"keyring", "carabiner"},
			priceInclTaxCents: 1300,
			stockQty:          150,
			ratingAverage:     4.4,
		},

		// Sunbeam Supply
		{
			ownerUserID:       "seed-owner-d",
			title:             "Glass Spice Jar Set",
			description:       "Four glass jars with minimalist labels and tight-seal lids.",
			categorySlug:      "kitchen",
			tags:              []string{"spices", "jars", "kitchen"},
			priceInclTaxCents: 4200,
			stockQty:          26,
			ratingAverage:     4.7,
		},
		{
			ownerUserID:       "seed-owner-d",
			title:             "Pour-Over Coffee Filters",
			description:       "Bright white paper filters sized for standard drippers.",
			categorySlug:      "kitchen",
			tags:              []string{"coffee", "filters"},
			priceInclTaxCents: 1100,
			stockQty:          140,
			ratingAverage:     4.5,
		},
		{
			ownerUserID:       "seed-owner-d",
			title:             "Maple Cutting Board",
			description:       "A durable cutting board with a soft matte finish.",
			categorySlug:      "kitchen",
			tags:              []string{"cutting-board", "maple", "kitchen"},
			priceInclTaxCents: 6800,
			stockQty:          15,
			ratingAverage:     4.8,
		},
		{
			ownerUserID:       "seed-owner-d",
			title:             "Soft Linen Napkins (2 pack)",
			description:       "Everyday linen napkins in a warm neutral tone.",
			categorySlug:      "home",
			tags:              []string{"linen", "napkins", "home"},
			priceInclTaxCents: 2600,
			stockQty:          62,
			ratingAverage:     4.6,
		},
		{
			ownerUserID:       "seed-owner-d",
			title:             "Soap Bar (Citrus + Cedar)",
			description:       "Simple, gentle soap with a fresh scent profile.",
			categorySlug:      "home",
			tags:              []string{"soap", "bath"},
			priceInclTaxCents: 900,
			stockQty:          190,
			ratingAverage:     4.4,
		},

		// Fern Atelier
		{
			ownerUserID:       "seed-owner-e",
			title:             "Relaxed Crew Tee",
			description:       "Soft cotton tee with a clean cut and minimal branding.",
			categorySlug:      "apparel",
			tags:              []string{"t-shirt", "cotton"},
			priceInclTaxCents: 4200,
			stockQty:          38,
			ratingAverage:     4.5,
		},
		{
			ownerUserID:       "seed-owner-e",
			title:             "Ribbed Beanie",
			description:       "Warm ribbed knit beanie for cold morning walks.",
			categorySlug:      "apparel",
			tags:              []string{"beanie", "knit"},
			priceInclTaxCents: 2800,
			stockQty:          55,
			ratingAverage:     4.6,
		},
		{
			ownerUserID:       "seed-owner-e",
			title:             "Trail Socks (2 pack)",
			description:       "Cushioned socks built for long days on your feet.",
			categorySlug:      "outdoors",
			tags:              []string{"socks", "trail"},
			priceInclTaxCents: 2400,
			stockQty:          80,
			ratingAverage:     4.7,
		},
		{
			ownerUserID:       "seed-owner-e",
			title:             "Compact Rain Poncho",
			description:       "Lightweight poncho that packs down into a small pouch.",
			categorySlug:      "outdoors",
			tags:              []string{"rain", "poncho", "packable"},
			priceInclTaxCents: 3900,
			stockQty:          34,
			ratingAverage:     4.4,
		},
		{
			ownerUserID:       "seed-owner-e",
			title:             "Canvas Cap",
			description:       "A structured cap with subtle stitching and a clean profile.",
			categorySlug:      "accessories",
			tags:              []string{"cap", "hat"},
			priceInclTaxCents: 3100,
			stockQty:          48,
			ratingAverage:     4.3,
		},

		// Moss & Mortar
		{
			ownerUserID:       "seed-owner-f",
			title:             "Stoneware Planter",
			description:       "Small stoneware planter with drainage hole and saucer.",
			categorySlug:      "home",
			tags:              []string{"planter", "stoneware"},
			priceInclTaxCents: 5200,
			stockQty:          17,
			ratingAverage:     4.8,
		},
		{
			ownerUserID:       "seed-owner-f",
			title:             "Wool Picnic Blanket",
			description:       "A warm blanket with a simple stripe pattern for outdoor hangs.",
			categorySlug:      "outdoors",
			tags:              []string{"blanket", "wool", "picnic"},
			priceInclTaxCents: 8900,
			stockQty:          9,
			ratingAverage:     4.9,
		},
		{
			ownerUserID:       "seed-owner-f",
			title:             "Incense Cones (Sandalwood)",
			description:       "Gentle incense cones for a calm evening ritual.",
			categorySlug:      "home",
			tags:              []string{"incense", "sandalwood"},
			priceInclTaxCents: 1600,
			stockQty:          105,
			ratingAverage:     4.5,
		},
		{
			ownerUserID:       "seed-owner-f",
			title:             "Plant Mister Spray Bottle",
			description:       "Fine-mist sprayer for keeping plants happy year-round.",
			categorySlug:      "home",
			tags:              []string{"plants", "mister"},
			priceInclTaxCents: 2300,
			stockQty:          46,
			ratingAverage:     4.4,
		},
		{
			ownerUserID:       "seed-owner-f",
			title:             "Leather Field Notebook Cover",
			description:       "A slim leather cover to protect your daily carry notebook.",
			categorySlug:      "accessories",
			tags:              []string{"leather", "notebook", "cover"},
			priceInclTaxCents: 7600,
			stockQty:          12,
			ratingAverage:     4.7,
		},
	}

	for _, p := range productsToSeed {
		vendorID, ok := vendorIDByOwner[p.ownerUserID]
		if !ok {
			continue
		}

		a.catalogService.CreateProductWithInput(catalog.CreateProductInput{
			OwnerUserID:       p.ownerUserID,
			VendorID:          vendorID,
			Title:             p.title,
			Description:       p.description,
			CategorySlug:      p.categorySlug,
			Tags:              p.tags,
			PriceInclTaxCents: p.priceInclTaxCents,
			Currency:          "USD",
			StockQty:          p.stockQty,
			RatingAverage:     p.ratingAverage,
			Status:            catalog.ProductStatusApproved,
		})
	}
}
