package home

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// SeasonOverrideKey is the app_config key of the Season Staff forced on Home.
const SeasonOverrideKey = "season_override"

// BannerTarget is a Banner one link: a Collection id, Section name, Book id or search query.
type BannerTarget struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Banner is BannerModel.
type Banner struct {
	ID         string       `json:"id"`
	TitleEn    string       `json:"titleEn"`
	TitleBn    string       `json:"titleBn"`
	SubtitleEn string       `json:"subtitleEn"`
	SubtitleBn string       `json:"subtitleBn"`
	Seed       int          `json:"seed"`
	Target     BannerTarget `json:"target"`
	Season     *string      `json:"season"`
}

// BannerFromRow converts a stored banner.
func BannerFromRow(b sqlc.Banner) Banner {
	out := Banner{ID: b.ID, TitleEn: b.TitleEn, TitleBn: b.TitleBn, SubtitleEn: b.SubtitleEn, SubtitleBn: b.SubtitleBn,
		Seed: int(b.Seed), Target: BannerTarget{Kind: b.TargetKind, Value: b.TargetValue}}
	if b.Season.Valid {
		s := b.Season.String
		out.Season = &s
	}
	return out
}

// Ayah is AyahModel.
type Ayah struct {
	Arabic      string `json:"arabic"`
	Translation string `json:"translation"`
	SurahEn     string `json:"surahEn"`
	SurahBn     string `json:"surahBn"`
	SurahNumber int    `json:"surahNumber"`
	VerseNumber int    `json:"verseNumber"`
}

// Service answers the Home endpoints.
type Service struct {
	DB    *db.DB
	Clock clock.Clock
	Loc   *time.Location
}

// SeasonOverride reads the Season Staff forced on Home ("" = automatic, by date).
func SeasonOverride(ctx context.Context, q *sqlc.Queries) (Season, error) {
	raw, err := q.GetConfig(ctx, SeasonOverrideKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var v struct {
		Season *string `json:"season"`
	}
	if err := json.Unmarshal(raw, &v); err != nil || v.Season == nil {
		return "", nil
	}
	return Season(*v.Season), nil
}

// Active is the Season on for the given day ("" for none).
func (s *Service) Active(ctx context.Context, day time.Time) (Season, error) {
	override, err := SeasonOverride(ctx, s.DB.Q())
	if err != nil {
		return "", err
	}
	return ActiveOn(day, override), nil
}

// Today is the current calendar day in the app timezone.
func (s *Service) Today() time.Time { return clock.Today(s.Clock, s.Loc) }

// Banners lists the active Season Banners first, then the all-year ones, in display order.
// Banners of other Seasons are left out.
func (s *Service) Banners(ctx context.Context, day time.Time) ([]Banner, error) {
	active, err := s.Active(ctx, day)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.Q().ListBanners(ctx)
	if err != nil {
		return nil, err
	}
	out := []Banner{}
	for _, r := range rows {
		if active != "" && r.Season.Valid && r.Season.String == string(active) {
			out = append(out, BannerFromRow(r))
		}
	}
	for _, r := range rows {
		if !r.Season.Valid {
			out = append(out, BannerFromRow(r))
		}
	}
	return out, nil
}

// Season returns the hero card of the Season on, or nil.
func (s *Service) Season(ctx context.Context, day time.Time) (*SeasonInfo, error) {
	active, err := s.Active(ctx, day)
	if err != nil || active == "" {
		return nil, err
	}
	info, ok := Info[active]
	if !ok {
		return nil, nil
	}
	return &info, nil
}

// AyahOfTheDay rotates through the verses by day of the year (January 1 is day 0).
func (s *Service) AyahOfTheDay(ctx context.Context, day time.Time) (*Ayah, error) {
	rows, err := s.DB.Q().ListAyahs(ctx)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	r := rows[(day.YearDay()-1)%len(rows)]
	return &Ayah{Arabic: r.Arabic, Translation: r.Translation, SurahEn: r.SurahEn, SurahBn: r.SurahBn,
		SurahNumber: int(r.SurahNumber), VerseNumber: int(r.VerseNumber)}, nil
}
