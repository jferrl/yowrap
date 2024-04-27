// Package yowrap provides a wrapper around the generated Yo model.
package yowrap

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
)

// ErrNoClient is returned when the spanner client is not set.
var ErrNoClient = errors.New("no spanner client")

// YoModel is the interface that wraps the basic methods of the Yo
// generated model.
type YoModel interface {
	Insert(ctx context.Context) *spanner.Mutation
	Update(ctx context.Context) *spanner.Mutation
	InsertOrUpdate(ctx context.Context) *spanner.Mutation
	Delete(ctx context.Context) *spanner.Mutation
}

// Opt is a function that modifies a Model.
type Opt[T YoModel] func(*Model[T])

// WithSpannerClientOption returns an Opt that sets the spanner client.
func WithSpannerClientOption[T YoModel](c spanner.Client) Opt[T] {
	return func(m *Model[T]) {
		m.Client = c
	}
}

// Model is a struct that embeds the generated model and implements the
// YoModel interface.
type Model[T YoModel] struct {
	YoModel
	spanner.Client

	hooks map[Hook][]HookFunc
}

// NewModel returns a new wrapped yo model.
func NewModel[T YoModel](m T, opts ...Opt[T]) *Model[T] {
	mo := &Model[T]{
		YoModel: m,
		hooks:   make(map[Hook][]HookFunc),
	}
	for _, opt := range opts {
		opt(mo)
	}

	return mo
}

// On registers an action to be executed before or after a method.
func (m *Model[T]) On(h Hook, f HookFunc) {
	m.hooks[h] = append(m.hooks[h], f)
}

// ApplyInsert inserts the model and applies the mutation.
func (m *Model[T]) ApplyInsert(ctx context.Context) (time.Time, error) {
	return m.readWriteTxn(ctx, Insert)
}

// ApplyUpdate updates the model and applies the mutation.
func (m *Model[T]) ApplyUpdate(ctx context.Context) (time.Time, error) {
	return m.readWriteTxn(ctx, Update)
}

// ApplyInsertOrUpdate inserts or updates the model and applies the mutation.
func (m *Model[T]) ApplyInsertOrUpdate(ctx context.Context) (time.Time, error) {
	return m.readWriteTxn(ctx, InsertOrUpdate)
}

// ApplyDelete deletes the model and applies the mutation.
func (m *Model[T]) ApplyDelete(ctx context.Context) (time.Time, error) {
	return m.readWriteTxn(ctx, Delete)
}

func (m *Model[T]) readWriteTxn(ctx context.Context, h Hook) (time.Time, error) {
	return m.Client.ReadWriteTransaction(ctx, func(ctx context.Context, rwt *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation

		if actions, ok := m.hooks[h]; ok {
			for _, f := range actions {
				mutations = append(mutations, f(ctx)...)
			}
		}

		mutations = append(mutations, m.Insert(ctx))

		return rwt.BufferWrite(mutations)
	})
}
