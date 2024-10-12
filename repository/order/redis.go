package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/model"
)

type RedisRepo struct {
	Client *redis.Client
}

func (r *RedisRepo) Insert(ctx context.Context, order model.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to encode order: %w", err)
	}

	// Generating unique key
	key := orderIDKey(order.OrderID)

	// Using transaction to make changes atomic
	txn := r.Client.TxPipeline()

	res := txn.SetNX(ctx, key, string(data), 0)
	if err := res.Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to insert order: %w", err)
	}

	if err := txn.SAdd(ctx, "orders", key).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to add orders set: %w", err)
	}

	if _, err := txn.Exec(ctx); err != nil {
		return fmt.Errorf("failed to exec: %w", err)
	}

	return nil
}

func orderIDKey(id uint64) string {
	return fmt.Sprintf("order:%d", id)
}

var ErrNotExist = errors.New("order does not exist")

func (r *RedisRepo) FindByID(ctx context.Context, id uint64) (model.Order, error) {

	// Generating unique key
	key := orderIDKey(id)
	value, err := r.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return model.Order{}, ErrNotExist
	} else if err != nil {
		return model.Order{}, fmt.Errorf("failed to insert order: %w", err)
	}

	var order model.Order
	err = json.Unmarshal([]byte(value), &order)
	if err != nil {
		return model.Order{}, fmt.Errorf("failed to decode order json: %w", err)
	}
	return order, nil
}

func (r *RedisRepo) DeleteByID(ctx context.Context, id uint64) error {
	// Generating unique key
	key := orderIDKey(id)

	txn := r.Client.TxPipeline()

	err := txn.Del(ctx, key).Err()
	if errors.Is(err, redis.Nil) {
		txn.Discard()
		return ErrNotExist
	} else if err != nil {
		txn.Discard()
		return fmt.Errorf("failed to delete order: %w", err)
	}

	if err := txn.SRem(ctx, "orders", key).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to remove orders set: %w", err)
	}

	if _, err := txn.Exec(ctx); err != nil {
		return fmt.Errorf("failed to exec: %w", err)
	}
	return nil
}

func (r *RedisRepo) Update(ctx context.Context, order model.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to encode order: %w", err)
	}
	// Generating unique key
	key := orderIDKey(order.OrderID)
	err = r.Client.SetXX(ctx, key, string(data), 0).Err()
	if errors.Is(err, redis.Nil) {
		return ErrNotExist
	} else if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	return nil
}

// Adding pagination~
type FindAllPage struct {
	Size   uint64
	Offset uint64
}

type FindResult struct {
	Orders []model.Order
	Cursor uint64
}

func (r *RedisRepo) FindAll(ctx context.Context, page FindAllPage) (FindResult, error) {
	res := r.Client.SScan(ctx, "orders", page.Offset, "*", int64(page.Size))

	keys, cursor, err := res.Result()
	if err != nil {
		return FindResult{}, fmt.Errorf("failed to get order ids: %w", err)
	}

	if len(keys) == 0 {
		return FindResult{
			Orders: []model.Order{},
		}, nil
	}

	xs, err := r.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return FindResult{}, fmt.Errorf("failed to get orders: %w", err)
	}

	orders := make([]model.Order, len(xs))

	for i, x := range xs {
		x := x.(string)
		var order model.Order

		err := json.Unmarshal([]byte(x), &order)
		if err != nil {
			return FindResult{}, fmt.Errorf("failed to decode order json: %w", err)
		}

		orders[i] = order
	}

	return FindResult{
		Orders: orders,
		Cursor: cursor,
	}, nil
}
