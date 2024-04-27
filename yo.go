package yowrap

import (
	"context"

	"cloud.google.com/go/spanner"
)

// YoModel is the interface that wraps the basic methods of the Yo
// generated model.
type YoModel interface {
	Insert(ctx context.Context) *spanner.Mutation
	Update(ctx context.Context) *spanner.Mutation
	InsertOrUpdate(ctx context.Context) *spanner.Mutation
	Delete(ctx context.Context) *spanner.Mutation
}
