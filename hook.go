package yowrap

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Hook defines an action that can be executed during a mutation.
type Hook int

const (
	// AfterInsert is executed when the Insert txn is called.
	AfterInsert Hook = iota + 1
	// AfterUpdate is executed when the Update txn is called.
	AfterUpdate
	// AfterInsertOrUpdate is executed when the InsertOrUpdate txn is called.
	AfterInsertOrUpdate
	// AfterDelete is executed when the Delete txn is called.
	AfterDelete
	// BeforeInsert is executed before the Insert txn is called.
	BeforeInsert
	// BeforeUpdate is executed before the Update txn is called.
	BeforeUpdate
	// BeforeInsertOrUpdate is executed before the InsertOrUpdate txn is called.
	BeforeInsertOrUpdate
	// BeforeDelete is executed before the Delete txn is called.
	BeforeDelete
)

// HookFunc defines a function that can be executed during a mutation.
// Is invoked with the model that is being mutated.
type HookFunc[T Yo] func(context.Context, *Model[T], *spanner.ReadWriteTransaction) error
