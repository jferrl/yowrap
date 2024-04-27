package yowrap

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Hook defines an action that can be executed before or after a method.
type Hook int

const (
	// Insert is executed when the Insert method is called.
	Insert Hook = iota + 1
	// Update is executed when the Update method is called.
	Update
	// InsertOrUpdate is executed when the InsertOrUpdate method is called.
	InsertOrUpdate
	// Delete is executed when the Delete method is called.
	Delete
)

// HookFunc is a trigger function associated with a kind of mutation.
// It executed to generate the mutations to be applied.
type HookFunc func(context.Context) []*spanner.Mutation
