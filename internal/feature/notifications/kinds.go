// Package notifications stores what happened to a reader and tells the app. It stores the kind
// and its params, never the words: the app builds the text from its ARB files.
package notifications

// Kind is NotificationKind in the app. Offers and messages are never notifications.
type Kind string

// The kinds, named as the Dart enum values.
const (
	KindOrderStatus       Kind = "orderStatus"
	KindReturnDecided     Kind = "returnDecided"
	KindListingDecided    Kind = "listingDecided"
	KindModerationWarning Kind = "moderationWarning"
	KindBanned            Kind = "banned"
	KindSaleSent          Kind = "saleSent"
	KindSaleCompleted     Kind = "saleCompleted"
	KindSaleSettled       Kind = "saleSettled"
	KindSellBackPaid      Kind = "sellBackPaid"
	KindSellBackReturned  Kind = "sellBackReturned"
	KindAlertTriggered    Kind = "alertTriggered"
	KindBookWanted        Kind = "bookWanted"
	KindNewFollower       Kind = "newFollower"
	KindBiteComment       Kind = "biteComment"
	KindCommentReply      Kind = "commentReply"
)

// Settings groups a reader can mute (NotificationGroup in the app).
const (
	GroupOrders    = "orders"
	GroupUsedBooks = "usedBooks"
	GroupAlerts    = "alerts"
	GroupCommunity = "community"
)

// Group is the settings switch that mutes this kind; "" for moderation, which cannot be muted.
// Port of NotificationKindX.group.
func (k Kind) Group() string {
	switch k {
	case KindOrderStatus, KindReturnDecided:
		return GroupOrders
	case KindModerationWarning, KindBanned:
		return ""
	case KindAlertTriggered:
		return GroupAlerts
	case KindNewFollower, KindBiteComment, KindCommentReply:
		return GroupCommunity
	}
	return GroupUsedBooks
}

// TargetKind is where tapping a notification goes (NotificationTargetKind in the app).
type TargetKind string

// The target kinds.
const (
	TargetOrder      TargetKind = "order"
	TargetListing    TargetKind = "listing"
	TargetMyListings TargetKind = "myListings"
	TargetSale       TargetKind = "sale"
	TargetSellBack   TargetKind = "sellBack"
	TargetBook       TargetKind = "book"
	TargetBite       TargetKind = "bite"
	TargetReader     TargetKind = "reader"
)

// Target is NotificationTargetModel.
type Target struct {
	Kind TargetKind `json:"kind"`
	ID   string     `json:"id"`
}

// To makes a target for kind, with an id where the page needs one.
func To(kind TargetKind, id string) *Target { return &Target{Kind: kind, ID: id} }
