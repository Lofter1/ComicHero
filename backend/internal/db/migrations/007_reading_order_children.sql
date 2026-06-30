-- +goose Up
CREATE TABLE reading_order_children (
    parent_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    child_reading_order_id  INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
    position                INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (parent_reading_order_id, child_reading_order_id),
    CHECK (parent_reading_order_id <> child_reading_order_id)
);

-- +goose Down
DROP TABLE reading_order_children;
