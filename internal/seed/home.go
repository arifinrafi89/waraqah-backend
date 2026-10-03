package seed

import (
	"context"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadHome loads Home: the Banners in display order, the verses of Ayah of the Day, and the
// outside books the ISBN lookup knows.
func loadHome(ctx context.Context, r *Run) error {
	var banners []struct {
		ID, TitleEn, TitleBn, SubtitleEn, SubtitleBn string
		Seed                                         int
		Target                                       struct{ Kind, Value string }
		Season                                       *string
	}
	var ayahs []struct {
		Arabic, Translation, SurahEn, SurahBn string
		SurahNumber, VerseNumber              int
	}
	var isbns []struct {
		Isbn, Title, Author, Publisher string
		TitleBn, Language, Format      *string
		ListPriceBdt                   int
	}
	for name, v := range map[string]any{"banners.json": &banners, "ayahs.json": &ayahs, "isbn_lookup.json": &isbns} {
		if err := r.read(name, v); err != nil {
			return err
		}
	}
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for i, b := range banners {
			err := q.UpsertSeedBanner(ctx, sqlc.UpsertSeedBannerParams{ID: b.ID, Position: int32(i + 1), TitleEn: b.TitleEn, TitleBn: b.TitleBn,
				SubtitleEn: b.SubtitleEn, SubtitleBn: b.SubtitleBn, Seed: int32(b.Seed), TargetKind: b.Target.Kind,
				TargetValue: b.Target.Value, Season: opt(b.Season)})
			if err != nil {
				return err
			}
		}
		for i, a := range ayahs {
			err := q.UpsertSeedAyah(ctx, sqlc.UpsertSeedAyahParams{Position: int32(i), Arabic: a.Arabic, Translation: a.Translation,
				SurahEn: a.SurahEn, SurahBn: a.SurahBn, SurahNumber: int32(a.SurahNumber), VerseNumber: int32(a.VerseNumber)})
			if err != nil {
				return err
			}
		}
		for _, b := range isbns {
			lang, format := "english", "paperback"
			if b.Language != nil {
				lang = *b.Language
			}
			if b.Format != nil {
				format = *b.Format
			}
			err := q.UpsertSeedIsbn(ctx, sqlc.UpsertSeedIsbnParams{Isbn: b.Isbn, Title: b.Title, TitleBn: opt(b.TitleBn), Author: b.Author,
				Publisher: b.Publisher, Language: lang, Format: format, ListPriceBdt: int32(b.ListPriceBdt)})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
