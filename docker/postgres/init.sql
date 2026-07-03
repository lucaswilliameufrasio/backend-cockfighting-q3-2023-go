CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE UNLOGGED TABLE IF NOT EXISTS people (
    id uuid PRIMARY KEY,
    nickname VARCHAR(32) UNIQUE NOT NULL,
    "name" VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    stack VARCHAR(32)[] DEFAULT '{}' NOT NULL,
    searchable text
);

CREATE OR REPLACE FUNCTION people_update_searchable() RETURNS trigger AS $$
BEGIN
    NEW.searchable := LOWER(NEW.name || ' ' || NEW.nickname || ' ' || array_to_string(NEW.stack, ' '));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER people_searchable_trigger
BEFORE INSERT OR UPDATE ON people
FOR EACH ROW EXECUTE FUNCTION people_update_searchable();

CREATE INDEX CONCURRENTLY IF NOT EXISTS people_search_idx ON people USING GIN(searchable gin_trgm_ops);
