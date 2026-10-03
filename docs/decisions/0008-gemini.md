# Gemini client

**What:** The assistant asks Gemini to word replies (T05, T18). Decision: the real client calls the Gemini REST API with net/http behind the gemini.Client interface, so no SDK (google.golang.org/genai) is added.

**Why not the standard library:** Not applicable: no dependency added. Revisit if the REST surface grows.
