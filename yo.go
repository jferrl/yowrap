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

// Yo is the common interface for the generated model.
type Yo interface {
	Insert(ctx context.Context) *spanner.Mutation
	Update(ctx context.Context) *spanner.Mutation
	InsertOrUpdate(ctx context.Context) *spanner.Mutation
	Delete(ctx context.Context) *spanner.Mutation
}

// Opt is a function that modifies a Model.
type Opt[T Yo] func(*Model[T])

// WithSpannerClientOption returns an Opt that sets the spanner client.
func WithSpannerClientOption[T Yo](c spanner.Client) Opt[T] {
	return func(m *Model[T]) {
		m.Client = c
	}
}

// Model is a struct that embeds the generated model and implements the
// YoModel interface.
type Model[T Yo] struct {
	Yo
	spanner.Client

	hooks map[Hook]HookFunc[T]
}

// NewModel returns a new wrapped yo model.
func NewModel[T Yo](m T, opts ...Opt[T]) *Model[T] {
	mo := &Model[T]{
		Yo:    m,
		hooks: make(map[Hook]HookFunc[T]),
	}
	for _, opt := range opts {
		opt(mo)
	}

	return mo
}

// On registers an action to be executed before or after a method.
func (m *Model[T]) On(h Hook, f HookFunc[T]) {
	m.hooks[h] = f
}

// Apply executes the mutation against the database.
func (m *Model[T]) Apply(ctx context.Context, mtype Mutation) (time.Time, error) {
	return m.Client.ReadWriteTransaction(ctx, func(ctx context.Context, rwt *spanner.ReadWriteTransaction) error {
		before, after := mtype.Hooks()

		if f, ok := m.hooks[before]; ok {
			if err := f(ctx, m, rwt); err != nil {
				return err
			}
		}

		var mut *spanner.Mutation
		switch mtype {
		case Insert:
			mut = m.Insert(ctx)
		case Update:
			mut = m.Update(ctx)
		case InsertOrUpdate:
			mut = m.InsertOrUpdate(ctx)
		case Delete:
			mut = m.Delete(ctx)
		default:
			return errors.New("unknown mutation type")
		}

		if err := rwt.BufferWrite([]*spanner.Mutation{mut}); err != nil {
			return err
		}

		if f, ok := m.hooks[after]; ok {
			if err := f(ctx, m, rwt); err != nil {
				return err
			}
		}

		return nil
	})
}
