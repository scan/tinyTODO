package graphql

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/scan/tinyTODO/model"
	"github.com/scan/tinyTODO/repo"
)

const allIDsKey = "todos:items:all"
const itemIDKey = "todos:items:%s"

func keyForID(id string) string {
	return fmt.Sprintf(itemIDKey, id)
}

type Resolver struct {
	logger *zap.Logger
	repo   repo.Repository
}

func NewResolver(logger *zap.Logger, repo repo.Repository) *Resolver {
	return &Resolver{
		logger: logger,
		repo:   repo,
	}
}

func (r *mutationResolver) AddItem(ctx context.Context, newItem NewItem) (*model.Item, error) {
	item := model.Item{
		ID:        uuid.New().String(),
		Title:     newItem.Title,
		Content:   newItem.Content,
		CreatedAt: time.Now(),
	}

	if err := r.repo.InsertItem(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *mutationResolver) RemoveItem(ctx context.Context, id string) (bool, error) {
	if err := r.repo.RemoveItem(id); err != nil {
		return false, err
	}

	return true, nil
}

func (r *queryResolver) Items(ctx context.Context, limit int, after *string) (*ItemConnection, error) {
	offset := 0
	before := time.Now()

	if after != nil {
		c, err := decodeCursor(*after)
		if err != nil {
			return nil, errors.Wrapf(err, "issue decoding the cursor")
		}

		offset = c.Start
		before = c.Before
	}

	items, err := r.repo.LoadItemsBefore(offset, limit, before)
	if err != nil {
		return nil, err
	}

	lastPage := len(items) < limit
	edges := make([]*ItemEdge, len(items))
	for i, item := range items {
		edges[i] = &ItemEdge{
			Node:   item,
			Cursor: cursor{Start: offset + i, Before: before}.String(),
		}
	}

	return &ItemConnection{
		Edges: edges,
		PageInfo: &PageInfo{
			HasPreviousPage: offset > 0,
			HasNextPage:     !lastPage,
			EndCursor:       cursor{Start: offset + limit, Before: before}.String(),
			StartCursor:     cursor{Start: offset, Before: before}.String(),
		},
	}, nil
}

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
