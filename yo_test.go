package yowrap

import (
	"context"
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

func TestModel_ApplyInsert(t *testing.T) {
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
		hooks   []HookFunc
		wantErr bool
		want    []*user.User
	}{
		{
			name:   "insert a new user into the database",
			fields: fields{model: model},
			want: []*user.User{
				{
					Name:  "John Doe",
					Email: "jdoe@email.com",
				},
			},
		},
		{
			name: "insert a new user into the database with a hook",
			fields: fields{
				model: model,
			},
			hooks: []HookFunc{
				func(ctx context.Context) []*spanner.Mutation {
					user := &user.User{
						ID:        uuid.NewString(),
						Name:      "Jane Doe",
						Email:     "jane.doe@email.com",
						CreatedAt: spanner.CommitTimestamp,
						UpdatedAt: spanner.CommitTimestamp,
					}

					return []*spanner.Mutation{
						user.Insert(ctx),
					}
				},
			},
			want: []*user.User{
				{
					Name:  "Jane Doe",
					Email: "jane.doe@email.com",
				},
				{
					Name:  "John Doe",
					Email: "jdoe@email.com",
				},
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

			for _, h := range tt.hooks {
				m.On(Insert, h)
			}

			if _, err := m.ApplyInsert(ctx); (err != nil) != tt.wantErr {
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
