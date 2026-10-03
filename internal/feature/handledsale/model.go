package handledsale

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Sale statuses, named as SaleStatus in the app. Waraqah holds the buyer's money from paid until
// the buyer confirms (completed) or a moderator settles a dispute (refunded or released).
const (
	Paid      = "paid"
	Sent      = "sent"
	Completed = "completed"
	Disputed  = "disputed"
	Refunded  = "refunded"
	Released  = "released"
	Cancelled = "cancelled"
)

// Steps a reader takes (SaleStep).
const (
	StepSend    = "send"
	StepCancel  = "cancel"
	StepConfirm = "confirm"
)

// methods are the prepaid PaymentMethod names: cash on delivery cannot be held by Waraqah.
var methods = []string{"bkash", "nagad", "card"}

// reasons are the DisputeReason names.
var reasons = []string{"notAsDescribed", "damaged", "photocopy", "wrongBook", "notReceived"}

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrListingUnavailable = Refusal(httpx.ErrListingUnavailable)
	ErrListingOwn         = Refusal(httpx.ErrListingOwn)
	ErrPrepaidOnly        = Refusal(httpx.ErrSalePrepaidOnly)
	ErrSaleUnknown        = Refusal(httpx.ErrSaleUnknown)
	ErrStepRefused        = Refusal(httpx.ErrSaleStepRefused)
	ErrDisputeInvalid     = Refusal(httpx.ErrDisputeInvalid)
	ErrPhotoInvalid       = Refusal(httpx.ErrPhotoInvalid)
	ErrNotDisputed        = Refusal(httpx.ErrSaleNotDisputed)
	ErrNothingToPayOut    = Refusal(httpx.ErrPayoutNothing)
)

// Sale is HandledSaleModel: the sale as one of its two readers sees it.
type Sale struct {
	ID            string    `json:"id"`
	ListingID     string    `json:"listingId"`
	Title         string    `json:"title"`
	Role          string    `json:"role"`
	OtherName     string    `json:"otherName"`
	PriceBdt      int       `json:"priceBdt"`
	DeliveryBdt   int       `json:"deliveryBdt"`
	FeeBdt        int       `json:"feeBdt"`
	Status        string    `json:"status"`
	Method        string    `json:"method"`
	CreatedAt     time.Time `json:"createdAt"`
	CoverSeed     int       `json:"coverSeed"`
	DisputeReason *string   `json:"disputeReason"`
	DisputeNote   *string   `json:"disputeNote"`
	DisputePhotos []string  `json:"disputePhotos"`
}

// Payout is PayoutModel.
type Payout struct {
	AmountBdt int       `json:"amountBdt"`
	At        time.Time `json:"at"`
}

// Earnings is EarningsModel: a seller's money from handled sales.
type Earnings struct {
	HeldBdt    int      `json:"heldBdt"`
	EarnedBdt  int      `json:"earnedBdt"`
	PaidOutBdt int      `json:"paidOutBdt"`
	Payouts    []Payout `json:"payouts"`
}

// Dispute is SaleDisputeModel: a disputed sale as a moderator sees it (from the buyer's side).
type Dispute struct {
	Sale       Sale   `json:"sale"`
	BuyerName  string `json:"buyerName"`
	SellerName string `json:"sellerName"`
}

// facts is what a sale needs from outside its row to be shown.
type facts struct {
	title     string
	coverSeed int
	names     map[string]string
}

// toSale builds the sale as viewer sees it (HandledSaleFakeStore.json).
func toSale(r sqlc.HandledSale, viewer string, f facts, loc *time.Location) Sale {
	role, other := "seller", r.BuyerID
	if r.BuyerID == viewer {
		role, other = "buyer", r.SellerID
	}
	name := f.names[other]
	if name == "" {
		name = "?"
	}
	s := Sale{ID: r.ID, ListingID: r.ListingID, Title: f.title, Role: role, OtherName: name, PriceBdt: int(r.PriceBdt),
		DeliveryBdt: int(r.DeliveryBdt), FeeBdt: int(r.FeeBdt), Status: r.Status, Method: r.Method,
		CreatedAt: r.CreatedAt.In(loc).Truncate(time.Second), CoverSeed: f.coverSeed, DisputePhotos: []string{}}
	if r.DisputeReason.Valid {
		s.DisputeReason = &r.DisputeReason.String
	}
	if r.DisputeNote.Valid {
		s.DisputeNote = &r.DisputeNote.String
	}
	_ = json.Unmarshal(r.DisputePhotos, &s.DisputePhotos)
	if s.DisputePhotos == nil {
		s.DisputePhotos = []string{}
	}
	return s
}

func buyerPays(r sqlc.HandledSale) int  { return int(r.PriceBdt + r.DeliveryBdt) }
func sellerGets(r sqlc.HandledSale) int { return int(r.PriceBdt - r.FeeBdt) }

func isPrepaid(method string) bool { return slices.Contains(methods, method) }
