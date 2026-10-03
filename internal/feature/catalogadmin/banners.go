package catalogadmin

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/home"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

var bannerTargets = []string{"collection", "section", "book", "search"}

func (s *Service) listBanners(ctx context.Context, q *sqlc.Queries) ([]home.Banner, error) {
	rows, err := q.ListBanners(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]home.Banner, 0, len(rows))
	for _, r := range rows {
		out = append(out, home.BannerFromRow(r))
	}
	return out, nil
}

// Banners lists every Banner, every Season Banners too, in display order.
func (s *Service) Banners(ctx context.Context) ([]home.Banner, error) {
	return s.listBanners(ctx, s.DB.Q())
}

func (s *Service) withBanners(ctx context.Context, fn func(q *sqlc.Queries) error) ([]home.Banner, error) {
	var out []home.Banner
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := fn(q); err != nil {
			return err
		}
		var err error
		out, err = s.listBanners(ctx, q)
		return err
	})
	return out, err
}

// SaveBanner is CatalogAdminFakeBanners.save: adds (empty id) or updates a Banner and answers them all.
func (s *Service) SaveBanner(ctx context.Context, b home.Banner) ([]home.Banner, error) {
	if len(CheckBanner(b.TitleEn, b.TitleBn, b.Target.Value)) > 0 || !oneOf(bannerTargets, b.Target.Kind) ||
		(b.Season != nil && !home.IsSeason(*b.Season)) {
		return nil, ErrBannerInvalid
	}
	season := pgtype.Text{}
	if b.Season != nil {
		season = pgtype.Text{String: *b.Season, Valid: true}
	}
	return s.withBanners(ctx, func(q *sqlc.Queries) error {
		if b.ID == "" {
			rows, err := q.ListBanners(ctx)
			if err != nil {
				return err
			}
			taken := make([]string, len(rows))
			for i, r := range rows {
				taken[i] = r.ID
			}
			pos, err := q.NextBannerPosition(ctx)
			if err != nil {
				return err
			}
			return q.InsertBanner(ctx, sqlc.InsertBannerParams{ID: uniqueID("ban", b.TitleEn, taken), Position: pos, TitleEn: strings.TrimSpace(b.TitleEn),
				TitleBn: strings.TrimSpace(b.TitleBn), SubtitleEn: b.SubtitleEn, SubtitleBn: b.SubtitleBn, Seed: int32(b.Seed),
				TargetKind: b.Target.Kind, TargetValue: b.Target.Value, Season: season})
		}
		if _, err := q.GetBanner(ctx, b.ID); errors.Is(err, pgx.ErrNoRows) {
			return ErrBannerUnknown
		} else if err != nil {
			return err
		}
		return q.UpdateBanner(ctx, sqlc.UpdateBannerParams{ID: b.ID, TitleEn: b.TitleEn, TitleBn: b.TitleBn, SubtitleEn: b.SubtitleEn,
			SubtitleBn: b.SubtitleBn, Seed: int32(b.Seed), TargetKind: b.Target.Kind, TargetValue: b.Target.Value, Season: season})
	})
}

// DeleteBanner is CatalogAdminFakeBanners.delete.
func (s *Service) DeleteBanner(ctx context.Context, id string) ([]home.Banner, error) {
	return s.withBanners(ctx, func(q *sqlc.Queries) error {
		n, err := q.DeleteBanner(ctx, id)
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrBannerUnknown
		}
		return nil
	})
}

// MoveBanner is CatalogAdminFakeBanners.move: one place up (by = -1) or down (1).
func (s *Service) MoveBanner(ctx context.Context, id string, by int) ([]home.Banner, error) {
	return s.withBanners(ctx, func(q *sqlc.Queries) error {
		rows, err := q.ListBanners(ctx)
		if err != nil {
			return err
		}
		from := -1
		for i, r := range rows {
			if r.ID == id {
				from = i
			}
		}
		to := from + by
		if from < 0 {
			return ErrBannerUnknown
		}
		if to < 0 || to >= len(rows) {
			return ErrBannerInvalid
		}
		order := make([]string, 0, len(rows))
		for _, r := range rows {
			if r.ID != id {
				order = append(order, r.ID)
			}
		}
		order = append(order[:to], append([]string{id}, order[to:]...)...)
		for i, bid := range order {
			if err := q.SetBannerPosition(ctx, sqlc.SetBannerPositionParams{ID: bid, Position: int32(i + 1)}); err != nil {
				return err
			}
		}
		return nil
	})
}

// SeasonOverride returns the Season Staff forced on Home ("" = automatic).
func (s *Service) SeasonOverride(ctx context.Context) (string, error) {
	v, err := home.SeasonOverride(ctx, s.DB.Q())
	return string(v), err
}

// SetSeasonOverride forces a Season on Home, or "" returns Home to automatic.
func (s *Service) SetSeasonOverride(ctx context.Context, season string) error {
	if season != "" && !home.IsSeason(season) {
		return ErrSeasonUnknown
	}
	value := []byte(`{"season":null}`)
	if season != "" {
		value = []byte(`{"season":"` + season + `"}`)
	}
	return s.DB.Q().SetConfig(ctx, sqlc.SetConfigParams{Key: home.SeasonOverrideKey, Value: value})
}
