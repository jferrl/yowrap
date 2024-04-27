package yowrap

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jferrl/yowrap/internal"
)

func TestModel_Insert(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		model *internal.User
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "yowrap model is able to expose the Insert method of the embedded model",
			fields: fields{model: &internal.User{
				ID:    uuid.NewString(),
				Name:  "John Doe",
				Email: "jdoe@email.com",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(tt.fields.model)

			if got := m.Insert(ctx) != nil; !got {
				t.Errorf("Model.Insert() = %v, want non-nil", got)
			}
		})
	}
}
