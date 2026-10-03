package checkout

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// CouponModel is the coupon as the app sends and receives it.
type CouponModel struct {
	Code           string     `json:"code"`
	Kind           string     `json:"kind"`
	Value          int        `json:"value"`
	MinOrderBdt    int        `json:"minOrderBdt"`
	MaxDiscountBdt *int       `json:"maxDiscountBdt"`
	ExpiresAt      *time.Time `json:"expiresAt"`
}

func toCouponModel(c sqlc.Coupon, loc *time.Location) CouponModel {
	out := CouponModel{Code: c.Code, Kind: c.Kind, Value: int(c.Value), MinOrderBdt: int(c.MinOrderBdt)}
	if c.MaxDiscountBdt.Valid {
		v := int(c.MaxDiscountBdt.Int32)
		out.MaxDiscountBdt = &v
	}
	if c.ExpiresAt.Valid {
		t := c.ExpiresAt.Time.In(loc).Truncate(time.Second)
		out.ExpiresAt = &t
	}
	return out
}

// Totals turns a coupon into what the totals need.
func (c CouponModel) Totals() Coupon {
	return Coupon{Kind: c.Kind, Value: c.Value, MinOrderBdt: c.MinOrderBdt, MaxDiscountBdt: c.MaxDiscountBdt}
}

// Expired reports whether the coupon stopped working: it works until its end date.
func (c CouponModel) Expired(now time.Time) bool {
	return c.ExpiresAt != nil && now.After(*c.ExpiresAt)
}

var codePattern = regexp.MustCompile(`^[A-Z0-9]{3,20}$`)

// CheckNewCoupon is the CreateCoupon rules of the app, enforced by the server:
//   - the code is 3 to 20 letters or digits (stored in capitals)
//   - percent off is 1 to 90 percent, amount off at least 1 taka
//   - the minimum order and the cap can not be negative
//   - an end date must be in the future
//
// It returns true when the coupon is acceptable.
func CheckNewCoupon(c CouponModel, now time.Time) bool {
	if !codePattern.MatchString(strings.ToUpper(strings.TrimSpace(c.Code))) {
		return false
	}
	switch c.Kind {
	case PercentOff:
		if c.Value < 1 || c.Value > 90 {
			return false
		}
	case AmountOff:
		if c.Value < 1 {
			return false
		}
	case FreeDelivery:
	default:
		return false
	}
	if c.MinOrderBdt < 0 || (c.MaxDiscountBdt != nil && *c.MaxDiscountBdt < 0) {
		return false
	}
	return c.ExpiresAt == nil || c.ExpiresAt.After(now)
}

// Coupons lists every coupon, newest first.
func (s *Service) Coupons(ctx context.Context) ([]CouponModel, error) {
	rows, err := s.DB.Q().ListCoupons(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CouponModel, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCouponModel(r, s.Loc))
	}
	return out, nil
}

// FindCoupon returns a coupon by code, expired or not, so checkout can say why; nil when unknown.
func (s *Service) FindCoupon(ctx context.Context, q *sqlc.Queries, code string) (*CouponModel, error) {
	row, err := q.GetCoupon(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m := toCouponModel(row, s.Loc)
	return &m, nil
}

// CreateCoupon adds a coupon (Staff) and answers the whole list. Refused when it breaks the rules
// or the code is taken.
func (s *Service) CreateCoupon(ctx context.Context, staffID string, in CouponModel) ([]CouponModel, error) {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	if !CheckNewCoupon(in, s.Clock.Now()) {
		return nil, ErrCouponInvalid
	}
	max := pgtype.Int4{}
	if in.MaxDiscountBdt != nil {
		max = pgtype.Int4{Int32: int32(*in.MaxDiscountBdt), Valid: true}
	}
	exp := pgtype.Timestamptz{}
	if in.ExpiresAt != nil {
		exp = pgtype.Timestamptz{Time: *in.ExpiresAt, Valid: true}
	}
	n, err := s.DB.Q().InsertCoupon(ctx, sqlc.InsertCouponParams{Code: in.Code, Kind: in.Kind, Value: int32(in.Value),
		MinOrderBdt: int32(in.MinOrderBdt), MaxDiscountBdt: max, ExpiresAt: exp, CreatedBy: pgtype.Text{String: staffID, Valid: staffID != ""}})
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrCouponTaken
	}
	return s.Coupons(ctx)
}
