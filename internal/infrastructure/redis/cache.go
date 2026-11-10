package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/mugishajc/orderforge/internal/domain"
)

const productCacheTTL = 15 * time.Minute

type ProductCache struct {
	client *goredis.Client
}

func NewProductCache(client *goredis.Client) *ProductCache {
	return &ProductCache{client: client}
}

func (c *ProductCache) Get(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	key := productKey(id)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var product domain.Product
	if err := json.Unmarshal(data, &product); err != nil {
		return nil, fmt.Errorf("unmarshal cached product: %w", err)
	}
	return &product, nil
}

func (c *ProductCache) Set(ctx context.Context, product *domain.Product) error {
	data, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("marshal product for cache: %w", err)
	}
	return c.client.Set(ctx, productKey(product.ID), data, productCacheTTL).Err()
}

func (c *ProductCache) Delete(ctx context.Context, id uuid.UUID) error {
	return c.client.Del(ctx, productKey(id)).Err()
}

func productKey(id uuid.UUID) string {
	return fmt.Sprintf("product:%s", id.String())
}
