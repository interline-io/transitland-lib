-- Index words with non-ASCII letters in tl text search, unaccented and lowercased.
--
-- Letter-only words were mapped only to unaccent, a filtering dictionary, so with nothing
-- after it they were dropped from every tsvector and tsquery; words mixing letters and
-- digits kept their accents. Stored textsearch columns keep their old vectors until
-- rewritten: see schema/postgres/ops/tl_text_search_unaccent.
ALTER TEXT SEARCH CONFIGURATION tl ALTER MAPPING FOR hword, hword_part, word, numword, numhword, hword_numpart WITH unaccent, simple;
