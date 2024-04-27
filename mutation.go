package yowrap

// Mutation defines a mutation that can be executed within spanner.
type Mutation int

const (
	// Insert is a mutation that inserts a row into a table.
	Insert Mutation = iota + 1
	// Update is a mutation that updates a row in a table.
	Update
	// InsertOrUpdate is a mutation that inserts a row into a table. If the row
	// already exists, it updates it instead. Any column values not explicitly
	// written are preserved.
	InsertOrUpdate
	// Delete is a mutation that deletes a row from a table.
	Delete
)

// Hooks defines an action that can be executed during a mutation.
func (m Mutation) Hooks() (before, after Hook) {
	switch m {
	case Insert:
		return BeforeInsert, AfterInsert
	case Update:
		return BeforeUpdate, AfterUpdate
	case InsertOrUpdate:
		return BeforeInsertOrUpdate, AfterInsertOrUpdate
	case Delete:
		return BeforeDelete, AfterDelete
	default:
		return 0, 0
	}
}
