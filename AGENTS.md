# Nestrr interface copy

- Keep visible static copy to the minimum necessary for using the interface.
- Do not add slogans, introductory copy, eyebrow labels, explanatory paragraphs, or decorative microcopy unless the user explicitly requests them.
- Prefer data from APIs or Nestrr's database over static descriptions.
- Static text is appropriate only when it is essential for an action, navigation, a concise state, validation, an error, safety, or accessibility.
- Empty states should identify the state concisely; do not explain what the user could do unless that instruction is required to proceed.
- Before adding visible copy, first try to communicate the same thing through layout, data, an established icon, or a short control label.

# Canonical TMDB media metadata

- Every row stored in Nestrr's canonical `movies` or `shows` catalog must come from TMDB's complete detail endpoint: `GET /3/movie/{movie_id}` for a movie or `GET /3/tv/{series_id}` for a show.
- Use `append_to_response` on those detail endpoints when Nestrr also needs related resources such as credits, keywords, release dates, or content ratings.
- Never write a movie or show listing result directly into the canonical catalog. Results from discover, search, popular, trending, recommendations, lists, or any other listing endpoint are partial candidate records only.
- Partial listing results may be stored only in a separate staging or work queue. Hydrate each candidate through its complete detail endpoint before inserting or updating the canonical `movies` or `shows` data.
