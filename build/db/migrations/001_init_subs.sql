CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL,
    monthly_fee INT NOT NULL,
    user_id UUID NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL
);

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE INDEX IF NOT EXISTS subs_period_gist
ON subscriptions
USING GIST (daterange(start_date, end_date, '[]'));

CREATE INDEX IF NOT EXISTS subs_order_idx
ON subscriptions (start_date, id);

CREATE INDEX IF NOT EXISTS subs_user_order_idx
ON subscriptions (user_id, start_date, id);

CREATE INDEX IF NOT EXISTS subs_service_order_idx
ON subscriptions (service_name, start_date, id);

---- create above / drop below ----

DROP INDEX IF EXISTS subs_service_order_idx;

DROP INDEX IF EXISTS subs_user_order_idx;

DROP INDEX IF EXISTS subs_order_idx;

DROP INDEX IF EXISTS subs_period_gist;

DROP EXTENSION IF EXISTS btree_gist;

DROP TABLE IF EXISTS subscriptions;
