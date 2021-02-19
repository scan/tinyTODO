package graphql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Resolver struct {
	logger *zap.Logger
	items  []*Item
}

func NewResolver(logger *zap.Logger) *Resolver {
	return &Resolver{
		logger: logger,
		items:  make([]*Item, 0),
	}
}

func (r *mutationResolver) AddItem(ctx context.Context, newItem NewItem) (bool, error) {
	item := Item{
		ID:        uuid.New().String(),
		Title:     newItem.Title,
		Content:   newItem.Content,
		CreatedAt: time.Now(),
	}

	r.items = append(r.items, &item)

	return true, nil
}

func (r *mutationResolver) RemoveItem(ctx context.Context, id string) (bool, error) {
	list := make([]*Item, 0, len(r.items))
	for _, item := range r.items {
		if item.ID != id {
			list = append(list, item)
		}
	}
	r.items = list

	return true, nil
}

func (r *queryResolver) Items(ctx context.Context, first, limit int) (*ItemConnection, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
