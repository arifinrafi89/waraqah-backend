// Package orders serves the orders of readers (list, details, cancel, return, buy again) and
// the Admin → Orders endpoints (move an order along, decide a return). Placing an order belongs
// to checkout; it saves through Insert.
package orders

import "time"

// Statuses, named as OrderStatus in the app.
const (
	Placed    = "placed"
	Confirmed = "confirmed"
	Packed    = "packed"
	Shipped   = "shipped"
	Delivered = "delivered"
	Cancelled = "cancelled"
)

// Steps are the tracking steps in order.
var Steps = []string{Placed, Confirmed, Packed, Shipped, Delivered}

// Next is the step after status, or "" once delivered or cancelled.
func Next(status string) string {
	for i, s := range Steps {
		if s == status && i < len(Steps)-1 {
			return Steps[i+1]
		}
	}
	return ""
}

// CanCancel: readers can cancel until the order leaves the warehouse.
func CanCancel(status string) bool {
	return status == Placed || status == Confirmed || status == Packed
}

// Return statuses, named as ReturnStatus in the app.
const (
	ReturnRequested = "requested"
	ReturnApproved  = "approved"
	ReturnRejected  = "rejected"
)

// MaxReturnPhotos is the most photos a return can carry.
const MaxReturnPhotos = 3

// ReturnWindow is how long after delivery a return can be asked for.
const ReturnWindow = 7 * 24 * time.Hour

// Line is OrderLineModel.
type Line struct {
	BookID       string  `json:"bookId"`
	Title        string  `json:"title"`
	Author       string  `json:"author"`
	Quantity     int     `json:"quantity"`
	UnitPriceBdt int     `json:"unitPriceBdt"`
	Format       *string `json:"format"`
	Language     *string `json:"language"`
	CoverSeed    int     `json:"coverSeed"`
	EditionID    *string `json:"editionId"`
}

// Change is StatusChangeModel: one step an order went through.
type Change struct {
	Status string    `json:"status"`
	At     time.Time `json:"at"`
}

// Return is ReturnRequestModel.
type Return struct {
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requestedAt"`
	Note        string    `json:"note"`
	Photos      []string  `json:"photos"`
}

// Gift is OrderGiftModel: who the order is for and the message on the card.
type Gift struct {
	RecipientName string `json:"recipientName"`
	Message       string `json:"message"`
	Wrapped       bool   `json:"wrapped"`
}

// Order is OrderModel: as every orders endpoint sends it.
type Order struct {
	Number         string    `json:"number"`
	PlacedAt       time.Time `json:"placedAt"`
	Status         string    `json:"status"`
	Lines          []Line    `json:"lines"`
	History        []Change  `json:"history"`
	AddressLabel   string    `json:"addressLabel"`
	AddressLine    string    `json:"addressLine"`
	Payment        string    `json:"payment"`
	SubtotalBdt    int       `json:"subtotalBdt"`
	DeliveryFeeBdt int       `json:"deliveryFeeBdt"`
	DiscountBdt    int       `json:"discountBdt"`
	TotalBdt       int       `json:"totalBdt"`
	NeedsDelivery  bool      `json:"needsDelivery"`
	PointsUsed     int       `json:"pointsUsed"`
	PointsEarned   int       `json:"pointsEarned"`
	ReturnRequest  *Return   `json:"returnRequest"`
	Gift           *Gift     `json:"gift"`
	GiftWrapBdt    int       `json:"giftWrapBdt"`
	IsDonation     bool      `json:"isDonation"`
	WalletUsedBdt  int       `json:"walletUsedBdt"`
	RefundedBdt    int       `json:"refundedBdt"`

	// UserID is the owner. It is never sent to the app.
	UserID string `json:"-"`
}

// ReachedAt is when the order reached a step, or nil.
func (o Order) ReachedAt(status string) *time.Time {
	for _, c := range o.History {
		if c.Status == status {
			t := c.At
			return &t
		}
	}
	return nil
}

// CanRequestReturn: delivered in the last 7 days, printed books, and not asked yet.
func (o Order) CanRequestReturn(now time.Time) bool {
	delivered := o.ReachedAt(Delivered)
	return o.NeedsDelivery && o.ReturnRequest == nil && delivered != nil && now.Sub(*delivered) <= ReturnWindow
}

// IsPrepaid: paid up front, as opposed to cash handed over at the door.
func IsPrepaid(payment string) bool { return payment != "cashOnDelivery" }

// CancelRefund is OrderRefunds.cancelRefundBdt: cancelling gives back what was paid: the wallet
// part always, and the rest only if it was paid up front (cash on delivery was never paid).
func (o Order) CancelRefund() int {
	if IsPrepaid(o.Payment) {
		return o.WalletUsedBdt + o.TotalBdt
	}
	return o.WalletUsedBdt
}

// ReturnRefund is OrderRefunds.returnRefundBdt: an approved return gives back what the books
// cost, whether paid by wallet, up front or at the door; delivery and gift wrap stay paid.
func (o Order) ReturnRefund() int {
	return max(0, o.TotalBdt+o.WalletUsedBdt-o.DeliveryFeeBdt-o.GiftWrapBdt)
}
