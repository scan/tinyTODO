package graphql

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Resolver struct {
	logger      *zap.Logger
	redisClient *redis.Client
}

func NewResolver(logger *zap.Logger, redisClient *redis.Client) *Resolver {
	return &Resolver{
		logger:      logger,
		redisClient: redisClient,
	}
}

func (r *mutationResolver) AddItem(ctx context.Context, newItem NewItem) (*Item, error) {
	item := Item{
		ID:        uuid.New().String(),
		Title:     newItem.Title,
		Content:   newItem.Content,
		CreatedAt: time.Now(),
	}

	err := r.redisClient.Watch(ctx, func(tx *redis.Tx) error {
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			data, err := item.asRedisEntry()
			if err != nil {
				return err
			}

			if err := r.redisClient.ZAdd(ctx, "todos:items:all", &redis.Z{
				Score:  float64(item.CreatedAt.UTC().Unix()),
				Member: item.ID,
			}).Err(); err != nil {
				return err
			}

			return r.redisClient.Set(ctx, fmt.Sprintf("todos:item:%s", item.ID), data, 0).Err()
		})

		return err
	})

	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *mutationResolver) RemoveItem(ctx context.Context, id string) (bool, error) {
	err := r.redisClient.Watch(ctx, func(tx *redis.Tx) error {
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			if err := r.redisClient.ZRem(ctx, "todos:items:all", id).Err(); err != nil {
				return err
			}

			return r.redisClient.Del(ctx, fmt.Sprintf("todos:item:%s", id)).Err()
		})

		return err
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *queryResolver) Items(ctx context.Context, first, limit int) (*ItemConnection, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
