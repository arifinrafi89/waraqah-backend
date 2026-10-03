package notifications

import (
	"context"
	"strconv"
)

// One function per event, ported from NotificationSends, NotificationSaleSends and
// NotificationCommunitySends. Each fills the params of its kind; the app writes the words.
// The first argument is always the Sender and the second the reader to tell.

func p(kv ...string) map[string]string {
	m := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

// OrderChanged tells the buyer the order moved to status.
func OrderChanged(ctx context.Context, s Sender, userID, number, status string) error {
	return s.Send(ctx, userID, KindOrderStatus, p("number", number, "status", status), To(TargetOrder, number))
}

// ReturnDecided tells the buyer a return was approved or rejected.
func ReturnDecided(ctx context.Context, s Sender, userID, number string, approved bool) error {
	return s.Send(ctx, userID, KindReturnDecided, p("number", number, "approved", strconv.FormatBool(approved)), To(TargetOrder, number))
}

// ListingDecided tells the seller what the moderator decided: approve, requestChanges or reject.
// An empty reason is left out of the params.
func ListingDecided(ctx context.Context, s Sender, sellerID, listingID, title, decision, reason string) error {
	params := p("title", title, "decision", decision)
	if reason != "" {
		params["reason"] = reason
	}
	return s.Send(ctx, sellerID, KindListingDecided, params, To(TargetListing, listingID))
}

// Warned sends a warning, or a ban once strikes reach max.
func Warned(ctx context.Context, s Sender, readerID string, strikes, max int) error {
	if strikes >= max {
		return BannedNotice(ctx, s, readerID)
	}
	return s.Send(ctx, readerID, KindModerationWarning, p("strikes", strconv.Itoa(strikes), "max", strconv.Itoa(max)), To(TargetMyListings, ""))
}

// BannedNotice tells a reader their account was banned.
func BannedNotice(ctx context.Context, s Sender, readerID string) error {
	return s.Send(ctx, readerID, KindBanned, p(), nil)
}

// AlertTriggered tells a reader a price or stock alert fired.
func AlertTriggered(ctx context.Context, s Sender, userID, bookID, title string, inStock bool) error {
	reason := "priceDrop"
	if inStock {
		reason = "backInStock"
	}
	return s.Send(ctx, userID, KindAlertTriggered, p("title", title, "reason", reason), To(TargetBook, bookID))
}

// WantedBook tells each seller who has a copy that a reader asked for title.
func WantedBook(ctx context.Context, s Sender, sellerIDs []string, title string) error {
	for _, id := range sellerIDs {
		if err := s.Send(ctx, id, KindBookWanted, p("title", title), To(TargetMyListings, "")); err != nil {
			return err
		}
	}
	return nil
}

// SaleSentNotice tells the buyer the seller sent the book.
func SaleSentNotice(ctx context.Context, s Sender, buyerID, saleID, title string) error {
	return s.Send(ctx, buyerID, KindSaleSent, p("title", title), To(TargetSale, saleID))
}

// SaleCompletedNotice tells the seller the buyer confirmed and the money is theirs.
func SaleCompletedNotice(ctx context.Context, s Sender, sellerID, saleID, title string, bdt int) error {
	return s.Send(ctx, sellerID, KindSaleCompleted, p("title", title, "amount", strconv.Itoa(bdt)), To(TargetSale, saleID))
}

// SaleSettledNotice tells both sides a moderator settled a dispute: refund the buyer or pay the
// seller. Each side hears with their own role.
func SaleSettledNotice(ctx context.Context, s Sender, buyerID, sellerID, saleID, title string, refund bool, amountBdt int) error {
	outcome := "paid"
	if refund {
		outcome = "refund"
	}
	for _, side := range [][2]string{{buyerID, "buyer"}, {sellerID, "seller"}} {
		params := p("title", title, "outcome", outcome, "role", side[1], "amount", strconv.Itoa(amountBdt))
		if err := s.Send(ctx, side[0], KindSaleSettled, params, To(TargetSale, saleID)); err != nil {
			return err
		}
	}
	return nil
}

// SellBackPaidNotice tells a reader Waraqah paid for their Sell Back.
func SellBackPaidNotice(ctx context.Context, s Sender, readerID, id, title string, bdt int) error {
	return s.Send(ctx, readerID, KindSellBackPaid, p("title", title, "amount", strconv.Itoa(bdt)), To(TargetSellBack, id))
}

// SellBackReturnedNotice tells a reader their Sell Back book is coming back.
func SellBackReturnedNotice(ctx context.Context, s Sender, readerID, id, title string) error {
	return s.Send(ctx, readerID, KindSellBackReturned, p("title", title), To(TargetSellBack, id))
}

// Followed tells a reader somebody started following them.
func Followed(ctx context.Context, s Sender, readerID, followerID, followerName string) error {
	return s.Send(ctx, readerID, KindNewFollower, p("name", followerName), To(TargetReader, followerID))
}

// BiteCommented tells the author somebody commented on their Bite.
func BiteCommented(ctx context.Context, s Sender, authorID, biteID, name, excerpt string) error {
	return s.Send(ctx, authorID, KindBiteComment, p("name", name, "excerpt", excerpt), To(TargetBite, biteID))
}

// CommentReplied tells the author somebody replied to their comment on a Bite.
func CommentReplied(ctx context.Context, s Sender, authorID, biteID, name string) error {
	return s.Send(ctx, authorID, KindCommentReply, p("name", name), To(TargetBite, biteID))
}
