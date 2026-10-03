# assistant

The AI assistant: picks Books from Waraqah's own catalog with rules ported from the app, and (optionally) lets Gemini word the reply around the Books already picked.

- **Frontend files:** `lib/features/ai_assistant/data/sources/{assistant_fake_api,assistant_brain,assistant_intent,assistant_parser,assistant_catalog,assistant_replies}.dart` (`intent.go`, `parser.go`, `picks.go`, `replies.go`, `brain.go`), with `ai_assistant_test.dart` and `smarter_ai_test.dart` ported.
- **Endpoints:** `GET /assistant/greeting?lang=` (public), `POST /assistant/ask` (me, rate-limited per reader by `AI_RATE_PER_MIN`). Answers `{id, text, bookIds}` and `basket: {editionIds, totalBdt}` for a basket request.
- **Catalog:** Books, Editions, prices and stock come from `catalog.Store.Snapshot`; hidden Books and Editions that cannot be ordered are never picked.
- **Gemini** (`GEMINI_API_KEY`, `GEMINI_MODEL`, `GEMINI_TIMEOUT`): only when Books were picked, asked for one or two sentences about exactly those Books; an error, timeout, empty or over-long answer keeps the rule-based text. Small talk and refusals never reach Gemini. Tests run without a key.
- **Tables:** none.
