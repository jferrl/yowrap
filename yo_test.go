package yowrap

import (
	"context"
	"errors"
	"os"
	"sort"
	"testing"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"cloud.google.com/go/spanner/spannertest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/jferrl/yowrap/internal/user"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestModel_Insert(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		model *user.User
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "yowrap model is able to expose the Insert method of the embedded model",
			fields: fields{model: &user.User{
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

func TestModel_Apply(t *testing.T) {
	ctx := context.Background()

	model := &user.User{
		ID:        uuid.NewString(),
		Name:      "John Doe",
		Email:     "jdoe@email.com",
		CreatedAt: spanner.CommitTimestamp,
		UpdatedAt: spanner.CommitTimestamp,
	}

	type fields struct {
		model *user.User
	}
	tests := []struct {
		name    string
		fields  fields
		mutType Mutation
		before  HookFunc[*user.User]
		after   HookFunc[*user.User]
		wantErr bool
		want    []*user.User
	}{
		{
			name:    "unknown mutation returns an error",
			fields:  fields{model: model},
			wantErr: true,
		},
		{
			name:    "insert a new user into the database",
			fields:  fields{model: model},
			mutType: Insert,
			want: []*user.User{
				{
					Name:  "John Doe",
					Email: "jdoe@email.com",
				},
			},
		},
		{
			name:    "insert or update user into the database",
			fields:  fields{model: model},
			mutType: InsertOrUpdate,
			want: []*user.User{
				{
					Name:  "John Doe",
					Email: "jdoe@email.com",
				},
			},
		},
		{
			name:    "update a non-existent user in the database",
			fields:  fields{model: model},
			mutType: Update,
			wantErr: true,
		},
		{
			name:    "before insert hook returns an error",
			fields:  fields{model: model},
			mutType: Insert,
			before: func(_ context.Context, m *Model[*user.User], _ *spanner.ReadWriteTransaction) error {
				return errors.New("before insert hook error")
			},
			wantErr: true,
		},
		{
			name:    "after insert hook returns an error",
			fields:  fields{model: model},
			mutType: Insert,
			after: func(_ context.Context, m *Model[*user.User], _ *spanner.ReadWriteTransaction) error {
				return errors.New("after insert hook error")
			},
			wantErr: true,
		},
		{
			name:    "insert a new user into the database with hooks",
			fields:  fields{model: model},
			mutType: Insert,
			before: func(_ context.Context, m *Model[*user.User], _ *spanner.ReadWriteTransaction) error {
				// type assertion to access the embedded model
				u, ok := m.Yo.(*user.User)
				if !ok {
					return errors.New("unable to type assert to *user.User")
				}

				u.Name = "Jane Doe"
				return nil
			},
			after: func(ctx context.Context, m *Model[*user.User], rwt *spanner.ReadWriteTransaction) error {
				// type assertion to access the embedded model
				u, ok := m.Yo.(*user.User)
				if !ok {
					return errors.New("unable to type assert to *user.User")
				}
				u.Email = "jane@gmail.com"

				return rwt.BufferWrite([]*spanner.Mutation{
					m.Update(ctx),
				})
			},
			want: []*user.User{
				{
					Name:  "Jane Doe",
					Email: "jane@gmail.com",
				},
			},
		},
		{
			name:    "delete a user from the database when is created with before delete hook",
			fields:  fields{model: model},
			mutType: Delete,
			before: func(cxt context.Context, m *Model[*user.User], txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite([]*spanner.Mutation{
					m.Insert(cxt),
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spannerClient, cleanup := setupSpanner(ctx, t)
			defer cleanup()

			m := NewModel(tt.fields.model,
				WithSpannerClientOption[*user.User](*spannerClient),
			)

			before, after := tt.mutType.Hooks()
			if tt.before != nil {
				m.On(before, tt.before)
			}
			if tt.after != nil {
				m.On(after, tt.after)
			}

			if _, err := m.Apply(ctx, tt.mutType); (err != nil) != tt.wantErr {
				t.Errorf("Model.ApplyInsert() error = %v, wantErr %v", err, tt.wantErr)
			}

			got, err := user.ReadUser(ctx, spannerClient.Single(), spanner.AllKeys())
			if err != nil {
				t.Fatal(err)
			}

			sort.Slice(got, func(i, j int) bool {
				return got[i].Name < got[j].Name
			})

			if diff := cmp.Diff(tt.want, got,
				cmpopts.IgnoreFields(user.User{}, "ID", "CreatedAt", "UpdatedAt")); diff != "" {
				t.Errorf("ApplyInsert() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func setupSpanner(ctx context.Context, t *testing.T) (*spanner.Client, func()) {
	server, err := spannertest.NewServer(":0")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := grpc.Dial(server.Addr, grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
	if err != nil {
		t.Fatal(err)
	}

	spannerDatabaseClient, err := database.NewDatabaseAdminClient(ctx, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}

	ddl, err := os.ReadFile("internal/sql/ddl.sql")
	if err != nil {
		t.Fatal(err)
	}

	op, err := spannerDatabaseClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   "my-db",
		Statements: []string{string(ddl)},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := op.Wait(ctx); err != nil {
		t.Fatal(err)
	}

	db := "projects/my-project/instances/my-instance/databases/my-db"
	spannerClient, err := spanner.NewClient(ctx, db, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}

	return spannerClient, func() {
		spannerClient.Close()
		spannerDatabaseClient.Close()
		server.Close()
	}
}
