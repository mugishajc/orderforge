-- OrderForge schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE customers (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email      VARCHAR(255) NOT NULL UNIQUE,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL,
    sku        VARCHAR(100) NOT NULL UNIQUE,
    price      BIGINT       NOT NULL CHECK (price >= 0),
    stock      INT          NOT NULL DEFAULT 0 CHECK (stock >= 0),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id  UUID         NOT NULL REFERENCES customers(id),
    status       VARCHAR(20)  NOT NULL DEFAULT 'draft',
    subtotal     BIGINT       NOT NULL DEFAULT 0,
    discount_amt BIGINT       NOT NULL DEFAULT 0,
    total        BIGINT       NOT NULL DEFAULT 0,
    coupon_code  VARCHAR(50),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status      ON orders(status);

CREATE TABLE order_items (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id   UUID   NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID   NOT NULL REFERENCES products(id),
    quantity   INT    NOT NULL CHECK (quantity > 0),
    unit_price BIGINT NOT NULL CHECK (unit_price >= 0),
    line_total BIGINT NOT NULL
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);

CREATE TABLE coupons (
    code          VARCHAR(50) PRIMARY KEY,
    discount_type VARCHAR(10) NOT NULL CHECK (discount_type IN ('fixed', 'percent')),
    value         BIGINT      NOT NULL CHECK (value >= 0),
    min_order_amt BIGINT      NOT NULL DEFAULT 0,
    max_uses      INT         NOT NULL DEFAULT 0,
    current_uses  INT         NOT NULL DEFAULT 0,
    expires_at    TIMESTAMPTZ NOT NULL
);
