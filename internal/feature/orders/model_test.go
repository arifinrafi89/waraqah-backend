package orders

import (
	"testing"
	"time"
)

// Ported from test/wallet_rules_test.dart (refund rules) and test/order_fake_store_test.dart (steps).
func TestRefundRules(t *testing.T) {
	o := Order{Payment: "cashOnDelivery", SubtotalBdt: 590, DeliveryFeeBdt: 60, TotalBdt: 550, WalletUsedBdt: 100}
	if got := o.ReturnRefund(); got != 590 {
		t.Errorf("a return refunds the books, not delivery: %d", got)
	}
	if got := o.CancelRefund(); got != 100 {
		t.Errorf("cash on delivery was never paid, only the wallet part comes back: %d", got)
	}
	o.Payment = "bkash"
	if got := o.CancelRefund(); got != 650 {
		t.Errorf("a prepaid order gives back the total and the wallet part: %d", got)
	}
	o.GiftWrapBdt = 40
	if got := o.ReturnRefund(); got != 550 {
		t.Errorf("gift wrap stays paid: %d", got)
	}
	if (Order{TotalBdt: 10, DeliveryFeeBdt: 60}).ReturnRefund() != 0 {
		t.Error("never negative")
	}
}

func TestStatusSteps(t *testing.T) {
	next := map[string]string{Placed: Confirmed, Confirmed: Packed, Packed: Shipped, Shipped: Delivered, Delivered: "", Cancelled: "", "weird": ""}
	for from, want := range next {
		if got := Next(from); got != want {
			t.Errorf("Next(%s) = %q want %q", from, got, want)
		}
	}
	for status, want := range map[string]bool{Placed: true, Confirmed: true, Packed: true, Shipped: false, Delivered: false, Cancelled: false} {
		if CanCancel(status) != want {
			t.Errorf("CanCancel(%s) = %v", status, !want)
		}
	}
}

func TestReturnWindow(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	delivered := func(ago time.Duration) Order {
		return Order{NeedsDelivery: true, History: []Change{{Status: Delivered, At: now.Add(-ago)}}}
	}
	if !delivered(48 * time.Hour).CanRequestReturn(now) {
		t.Error("two days after delivery")
	}
	if !delivered(7 * 24 * time.Hour).CanRequestReturn(now) {
		t.Error("exactly seven days still counts")
	}
	if delivered(7*24*time.Hour + time.Minute).CanRequestReturn(now) {
		t.Error("after seven days")
	}
	asked := delivered(time.Hour)
	asked.ReturnRequest = &Return{}
	if asked.CanRequestReturn(now) {
		t.Error("only once")
	}
	if (Order{NeedsDelivery: true}).CanRequestReturn(now) {
		t.Error("not delivered yet")
	}
	ebook := delivered(time.Hour)
	ebook.NeedsDelivery = false
	if ebook.CanRequestReturn(now) {
		t.Error("eBook-only orders cannot be returned")
	}
}
