package p2p

import (
	"context"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/cloudinary"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// SaveInput is the body of /p2p/listings/save: the form fields, the photo slots kept, the photos
// picked since ({slot: base64}) and whether to send it for review.
type SaveInput struct {
	ID           *string           `json:"id"`
	Title        string            `json:"title"`
	PriceBdt     int               `json:"priceBdt"`
	Condition    string            `json:"condition"`
	Flags        []string          `json:"flags"`
	Photos       []string          `json:"photos"`
	PhotoData    map[string]string `json:"photoData"`
	IsNegotiable bool              `json:"isNegotiable"`
	Handover     string            `json:"handover"`
	Submit       bool              `json:"submit"`
	BookID       *string           `json:"bookId"`
	NewPriceBdt  *int              `json:"newPriceBdt"`
	Note         *string           `json:"note"`
}

var (
	conditions = []string{"likeNew", "veryGood", "good", "acceptable"}
	handovers  = []string{"meetInPerson", "delivery"}
)

// byName is what the app does with an unknown enum value: the first one.
func byName(values []string, v string) string {
	if slices.Contains(values, v) {
		return v
	}
	return values[0]
}

// Save saves the listing of a reader from the add-listing form: a new one, or one they may still
// change (a draft, one sent back or rejected). A draft needs only its title; sending needs the
// price and the photos. Photos are uploaded here (and the assets of removed slots deleted).
func (s *Service) Save(ctx context.Context, userID string, in SaveInput) (*Listing, error) {
	if banned, err := s.Bans.IsBanned(ctx, userID); err != nil {
		return nil, err
	} else if banned {
		return nil, ErrBanned
	}
	q := s.DB.Q()
	var old *sqlc.Listing
	var oldSlots []string
	if in.ID != nil {
		l, err := q.LockListing(ctx, *in.ID)
		if isNoRows(err) || (err == nil && l.SellerID != userID) {
			return nil, ErrUnknown
		}
		if err != nil {
			return nil, err
		}
		if !CanEdit(l.Status) {
			return nil, ErrNotEditable
		}
		old = &l
		photos, err := q.ListPhotosOf(ctx, []string{l.ID})
		if err != nil {
			return nil, err
		}
		for _, p := range photos {
			oldSlots = append(oldSlots, p.Slot)
		}
	}

	images := map[string]cloudinary.Image{}
	for _, slot := range PhotoSlots {
		data, sent := in.PhotoData[slot]
		if !sent {
			continue
		}
		img, err := cloudinary.DecodeImage(data, s.MaxImageBytes)
		if err != nil {
			return nil, ErrPhotoInvalid
		}
		images[slot] = img
	}
	// A slot keeps its photo when one was just sent, or when the seller already had it and did not remove it.
	var final []string
	for _, slot := range PhotoSlots {
		if _, sent := images[slot]; sent || (slices.Contains(in.Photos, slot) && slices.Contains(oldSlots, slot)) {
			final = append(final, slot)
		}
	}
	var flags []string
	for _, f := range Flags {
		if slices.Contains(in.Flags, f) {
			flags = append(flags, f)
		}
	}
	note := ""
	if in.Note != nil {
		note = strings.TrimSpace(*in.Note)
	}
	title := strings.TrimSpace(in.Title)
	if Check(Fields{Title: title, Note: note, PriceBdt: in.PriceBdt, Flags: flags, Photos: final}, in.Submit) != ProblemNone {
		return nil, ErrInvalid
	}
	return s.write(ctx, userID, in, old, final, images, flags, title, note)
}

func (s *Service) write(ctx context.Context, userID string, in SaveInput, old *sqlc.Listing, final []string,
	images map[string]cloudinary.Image, flags []string, title, note string) (*Listing, error) {
	id := ids.Listing()
	if old != nil {
		id = old.ID
	}
	// Upload first: a failure here leaves the database untouched.
	uploaded := map[string]cloudinary.Result{}
	var fresh []string
	for _, slot := range PhotoSlots {
		img, ok := images[slot]
		if !ok {
			continue
		}
		res, err := s.Images.Upload(ctx, "listing", id, slot, img)
		if err != nil {
			s.destroy(ctx, fresh)
			return nil, err
		}
		uploaded[slot] = res
		fresh = append(fresh, res.PublicID)
	}
	status, reason := Draft, pgtype.Text{}
	if old != nil {
		status, reason = old.Status, old.RejectionReason
	}
	if in.Submit {
		status, reason = InReview, pgtype.Text{} // sending clears the answer of a moderator
	}
	opt := func(p *string) pgtype.Text {
		if p == nil || *p == "" {
			return pgtype.Text{}
		}
		return pgtype.Text{String: *p, Valid: true}
	}
	newPrice := pgtype.Int4{}
	if in.NewPriceBdt != nil {
		newPrice = pgtype.Int4{Int32: int32(*in.NewPriceBdt), Valid: true}
	}
	var removed []string
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if old == nil {
			who, err := q.GetPerson(ctx, userID)
			if err != nil {
				return err
			}
			seed := 0
			for _, b := range []byte(id) {
				seed += int(b)
			}
			err = q.InsertListing(ctx, sqlc.InsertListingParams{ID: id, SellerID: userID, Title: title, PriceBdt: int32(in.PriceBdt),
				Condition: byName(conditions, in.Condition), Flags: orEmpty(flags), IsNegotiable: in.IsNegotiable, Handover: byName(handovers, in.Handover),
				Status: status, RejectionReason: reason, BookID: opt(in.BookID), CoverSeed: int32(seed),
				District: pgtype.Text{String: who.District, Valid: true}, Area: pgtype.Text{String: who.Area, Valid: true},
				NewPriceBdt: newPrice, Note: opt(&note)})
			if err != nil {
				return err
			}
		} else {
			cur, err := q.LockListing(ctx, id)
			if err != nil {
				return err
			}
			if !CanEdit(cur.Status) {
				return ErrNotEditable
			}
			err = q.UpdateListing(ctx, sqlc.UpdateListingParams{ID: id, Title: title, PriceBdt: int32(in.PriceBdt),
				Condition: byName(conditions, in.Condition), Flags: orEmpty(flags), IsNegotiable: in.IsNegotiable, Handover: byName(handovers, in.Handover),
				Status: status, RejectionReason: reason, BookID: opt(in.BookID), NewPriceBdt: newPrice, Note: opt(&note)})
			if err != nil {
				return err
			}
		}
		gone, err := q.DeleteListingPhotosExcept(ctx, sqlc.DeleteListingPhotosExceptParams{ListingID: id, Slots: orEmpty(final)})
		if err != nil {
			return err
		}
		removed = append(removed, gone...)
		for _, slot := range final {
			res, ok := uploaded[slot]
			if !ok {
				continue
			}
			if err := q.UpsertListingPhoto(ctx, sqlc.UpsertListingPhotoParams{ListingID: id, Slot: slot,
				Position: int32(slices.Index(PhotoSlots, slot)), Url: res.URL, PublicID: res.PublicID}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if old == nil {
			s.destroy(ctx, fresh)
		}
		return nil, err
	}
	// Assets of removed slots go, unless a new photo took the same place.
	s.destroy(ctx, slices.DeleteFunc(removed, func(p string) bool { return p == "" || slices.Contains(fresh, p) }))
	return s.One(ctx, userID, id)
}

// destroy deletes Cloudinary assets, best effort.
func (s *Service) destroy(ctx context.Context, publicIDs []string) {
	for _, p := range publicIDs {
		if p == "" {
			continue
		}
		if err := s.Images.Destroy(ctx, p); err != nil {
			s.Log.Error("could not delete a listing photo", "public_id", p, "error", err)
		}
	}
}
