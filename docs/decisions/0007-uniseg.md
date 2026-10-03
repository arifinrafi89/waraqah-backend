# rivo/uniseg

**What:** Counts grapheme clusters so Bite length limits match Dart characters for Bangla and emoji (github.com/rivo/uniseg).

**Why not the standard library:** len() counts bytes and utf8.RuneCountInString counts code points; both are wrong for combined Bangla letters.
