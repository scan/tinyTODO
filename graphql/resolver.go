package graphql

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const allIDsKey = "todos:items:all"
const itemIDKey = "todos:items:%s"

func keyForID(id string) string {
	return fmt.Sprintf(itemIDKey, id)
}

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

			if err := r.redisClient.ZAdd(ctx, allIDsKey, &redis.Z{
				Score:  float64(item.CreatedAt.UTC().Unix()),
				Member: item.ID,
			}).Err(); err != nil {
				return err
			}

			return r.redisClient.Set(ctx, keyForID(item.ID), data, 0).Err()
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
			if err := r.redisClient.ZRem(ctx, allIDsKey, id).Err(); err != nil {
				return err
			}

			return r.redisClient.Del(ctx, keyForID(id)).Err()
		})

		return err
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *queryResolver) Items(ctx context.Context, first, limit int, after *string) (*ItemConnection, error) {
	last := int64(first + limit - 1)

	if after != nil {
		c, err := decodeCursor(*after)
		if err != nil {
			return nil, errors.Wrapf(err, "issue decoding the cursor")
		}

		first = c.Start + first
		if c.Limit > 0 {
			limit = c.Limit
		}
	}

	ids, err := r.redisClient.ZRange(ctx, allIDsKey, int64(first), last).Result()
	if err != nil {
		return nil, err
	}

	lastPage := len(ids) < limit

	r.logger.Info("Cursor info", zap.Int("id length", len(ids)), zap.Int("first", first), zap.Int("limit", limit))

	items := make([]Item, len(ids))
	for i, id := range ids {
		str, err := r.redisClient.Get(ctx, keyForID(id)).Result()
		if err != nil {
			return nil, err
		}

		if items[i], err = newItemFromRedisEntry([]byte(str)); err != nil {
			return nil, err
		}
	}

	edges := make([]*ItemEdge, len(items))
	for i, item := range items {
		edges[i] = &ItemEdge{
			Node:   &item,
			Cursor: cursor{Start: first + i, Limit: limit}.String(), // TODO
		}
	}

	return &ItemConnection{
		Edges: edges,
		PageInfo: &PageInfo{
			HasPreviousPage: first > 0,
			HasNextPage:     !lastPage,
			EndCursor:       cursor{Start: first + limit, Limit: limit}.String(),
			StartCursor:     cursor{Start: first, Limit: limit}.String(),
		},
	}, nil
}

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
