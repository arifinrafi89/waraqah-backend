package catalog

import "testing"

func edition(id, format string, stock int, preorder bool, price int) Edition {
	return Edition{ID: id, Format: format, Language: "english", PriceBdt: price, Stock: stock, IsPreorder: preorder}
}

// Ported from test/delivery_estimate_test.dart.
func TestDeliveryEstimates(t *testing.T) {
	paper := edition("e1", "paperback", 10, false, 500)
	if got := DeliveryEstimateOf(paper, InsideDhaka); got != (Estimate{ShipsInDays, 1, 2}) {
		t.Errorf("inside: %+v", got)
	}
	if got := DeliveryEstimateOf(paper, OutsideDhaka); got != (Estimate{ShipsInDays, 3, 5}) {
		t.Errorf("outside: %+v", got)
	}
	if got := DeliveryEstimateOf(edition("e", "ebook", 10, false, 1), OutsideDhaka); got.Kind != InstantDownload {
		t.Errorf("ebook: %+v", got)
	}
	if got := DeliveryEstimateOf(edition("e", "paperback", 0, true, 1), InsideDhaka); got.Kind != ShipsOnRelease {
		t.Errorf("preorder: %+v", got)
	}
	if got := DeliveryEstimateOf(edition("e", "paperback", 0, false, 1), InsideDhaka); got.Kind != Unavailable {
		t.Errorf("out of stock: %+v", got)
	}
}

func TestChosenEdition(t *testing.T) {
	b := Book{ID: "b1", Editions: []Edition{edition("cheap", "paperback", 10, false, 300), edition("pricey", "paperback", 10, false, 900)}}
	if b.ChosenEdition("").ID != "cheap" || b.ChosenEdition("pricey").ID != "pricey" || b.ChosenEdition("gone").ID != "cheap" {
		t.Error("chosen edition rules")
	}
}
