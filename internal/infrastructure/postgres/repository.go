package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mugishajc/orderforge/internal/domain"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, customer_id, status, subtotal, discount_amt, total, coupon_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		order.ID, order.CustomerID, order.Status,
		order.Subtotal, order.DiscountAmt, order.Total,
		order.CouponCode, order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	for _, item := range order.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, line_total)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			item.ID, item.OrderID, item.ProductID,
			item.Quantity, item.UnitPrice, item.LineTotal,
		)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	order := &domain.Order{}
	var couponCode *string

	err := r.pool.QueryRow(ctx, `
		SELECT id, customer_id, status, subtotal, discount_amt, total, coupon_code, created_at, updated_at
		FROM orders WHERE id = $1`, id,
	).Scan(
		&order.ID, &order.CustomerID, &order.Status,
		&order.Subtotal, &order.DiscountAmt, &order.Total,
		&couponCode, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		return nil, domain.ErrOrderNotFound
	}
	if couponCode != nil {
		order.CouponCode = *couponCode
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, order_id, product_id, quantity, unit_price, line_total
		FROM order_items WHERE order_id = $1`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("query order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.LineTotal); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}

func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	order.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = $1, subtotal = $2, discount_amt = $3, total = $4, coupon_code = $5, updated_at = $6
		WHERE id = $7`,
		order.Status, order.Subtotal, order.DiscountAmt, order.Total, order.CouponCode, order.UpdatedAt, order.ID,
	)
	return err
}

func (r *OrderRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.Order, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, customer_id, status, subtotal, discount_amt, total, coupon_code, created_at, updated_at
		FROM orders WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, customerID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		var couponCode *string
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.Status, &o.Subtotal, &o.DiscountAmt, &o.Total, &couponCode, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		if couponCode != nil {
			o.CouponCode = *couponCode
		}
		orders = append(orders, o)
	}

	return orders, nil
}

// ProductRepository implements domain.ProductRepository with PostgreSQL.
type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	p := &domain.Product{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, sku, price, stock, created_at, updated_at
		FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.SKU, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE products SET name = $1, sku = $2, price = $3, stock = $4, updated_at = $5
		WHERE id = $6`,
		product.Name, product.SKU, product.Price, product.Stock, product.UpdatedAt, product.ID,
	)
	return err
}

// CouponRepository implements domain.CouponRepository with PostgreSQL.
type CouponRepository struct {
	pool *pgxpool.Pool
}

func NewCouponRepository(pool *pgxpool.Pool) *CouponRepository {
	return &CouponRepository{pool: pool}
}

func (r *CouponRepository) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	c := &domain.Coupon{}
	err := r.pool.QueryRow(ctx, `
		SELECT code, discount_type, value, min_order_amt, max_uses, current_uses, expires_at
		FROM coupons WHERE code = $1`, code,
	).Scan(&c.Code, &c.Type, &c.Value, &c.MinOrderAmt, &c.MaxUses, &c.CurrentUses, &c.ExpiresAt)
	if err != nil {
		return nil, domain.ErrCouponNotFound
	}
	return c, nil
}

func (r *CouponRepository) IncrementUsage(ctx context.Context, code string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE coupons SET current_uses = current_uses + 1 WHERE code = $1`, code,
	)
	return err
}
