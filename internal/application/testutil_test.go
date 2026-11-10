package application_test

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/mugishajc/orderforge/internal/domain"
)

// In-memory test doubles

type memOrderRepo struct {
	mu     sync.Mutex
	orders map[uuid.UUID]*domain.Order
}

func newMemOrderRepo() *memOrderRepo {
	return &memOrderRepo{orders: make(map[uuid.UUID]*domain.Order)}
}

func (r *memOrderRepo) Create(_ context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *memOrderRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.orders[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

func (r *memOrderRepo) Update(_ context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *memOrderRepo) ListByCustomer(_ context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*domain.Order
	for _, o := range r.orders {
		if o.CustomerID == customerID {
			result = append(result, o)
		}
	}
	if offset >= len(result) {
		return nil, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

type memProductRepo struct {
	mu       sync.Mutex
	products map[uuid.UUID]*domain.Product
}

func newMemProductRepo() *memProductRepo {
	return &memProductRepo{products: make(map[uuid.UUID]*domain.Product)}
}

func (r *memProductRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[id]
	if !ok {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

func (r *memProductRepo) Update(_ context.Context, product *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[product.ID] = product
	return nil
}

func (r *memProductRepo) Add(p *domain.Product) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[p.ID] = p
}

type memCouponRepo struct {
	mu      sync.Mutex
	coupons map[string]*domain.Coupon
}

func newMemCouponRepo() *memCouponRepo {
	return &memCouponRepo{coupons: make(map[string]*domain.Coupon)}
}

func (r *memCouponRepo) GetByCode(_ context.Context, code string) (*domain.Coupon, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.coupons[code]
	if !ok {
		return nil, domain.ErrCouponNotFound
	}
	return c, nil
}

func (r *memCouponRepo) IncrementUsage(_ context.Context, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.coupons[code]; ok {
		c.CurrentUses++
	}
	return nil
}

func (r *memCouponRepo) Add(c *domain.Coupon) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.coupons[c.Code] = c
}

type noopCache struct{}

func (c *noopCache) Get(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
	return nil, domain.ErrProductNotFound
}
func (c *noopCache) Set(_ context.Context, _ *domain.Product) error { return nil }
func (c *noopCache) Delete(_ context.Context, _ uuid.UUID) error    { return nil }

type recordingPublisher struct {
	mu       sync.Mutex
	Messages []struct {
		Subject string
		Data    []byte
	}
}

func (p *recordingPublisher) Publish(_ context.Context, subject string, data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Messages = append(p.Messages, struct {
		Subject string
		Data    []byte
	}{subject, data})
	return nil
}
