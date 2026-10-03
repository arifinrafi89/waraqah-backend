package inbox

import "time"

// Offer is OfferModel.
type Offer struct {
	ID        string `json:"id"`
	AmountBdt int    `json:"amountBdt"`
	Handover  string `json:"handover"`
	Status    string `json:"status"`
}

// Message is InboxMessageModel: a chat message, an offer or a deal event, from the side of the
// viewer. Exactly one of text, offer and event is sent; the others are left out.
type Message struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	At        time.Time `json:"at"`
	Text      *string   `json:"text,omitempty"`
	Offer     *Offer    `json:"offer,omitempty"`
	Event     *string   `json:"event,omitempty"`
	AmountBdt *int      `json:"amountBdt,omitempty"`
}

// ThreadListing is ThreadListingModel: the listing as the thread shows it.
type ThreadListing struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	PriceBdt     int    `json:"priceBdt"`
	Status       string `json:"status"`
	CoverSeed    int    `json:"coverSeed"`
	IsNegotiable bool   `json:"isNegotiable"`
	Handover     string `json:"handover"`
}

// Thread is InboxThreadModel, as the signed-in reader sees it. The role and the unread count
// come from the token, never from the request.
type Thread struct {
	ID          string        `json:"id"`
	Role        string        `json:"role"`
	OtherID     string        `json:"otherId"`
	OtherName   string        `json:"otherName"`
	Listing     ThreadListing `json:"listing"`
	DealHere    bool          `json:"dealHere"`
	Unread      int           `json:"unread"`
	Messages    []Message     `json:"messages"`
	MyRating    *int          `json:"myRating"`
	TheirRating *int          `json:"theirRating"`

	lastAt time.Time
}
