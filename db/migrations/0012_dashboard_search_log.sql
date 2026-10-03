-- +goose Up
-- What readers search the catalog for, counted per term and Dhaka day, for the admin dashboard.
-- Staff searches with includeHidden are not counted; a live search that grows ("sap" → "sapiens")
-- counts once (search_log.dart).
CREATE TABLE search_log (
    term  text NOT NULL,
    day   date NOT NULL,
    count integer NOT NULL CHECK (count > 0),
    PRIMARY KEY (term, day)
);
CREATE INDEX search_log_day_idx ON search_log (day);

-- +goose Down
DROP TABLE IF EXISTS search_log;
