package text2sql

import (
	"context"
	"testing"

	"text2sql/internal/llm"
)

type mockProvider struct{}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) Complete(ctx context.Context, req *llm.CompleteRequest) (*llm.CompleteResponse, error) {
	return &llm.CompleteResponse{
		Content: "SELECT * FROM users\n解释：查询所有用户",
	}, nil
}

func TestService_Generate_NewConversation(t *testing.T) {
	provider := &mockProvider{}
	validator := NewSQLValidator()
	store := NewMemoryContextStore()
	svc := NewServiceWithContextStore(provider, validator, 2, store)

	req := &GenerateRequest{
		Query: "查询所有用户",
		Schema: Schema{
			Tables: []Table{
				{
					Name: "users",
					Columns: []Column{
						{Name: "id", Type: "int"},
						{Name: "name", Type: "varchar(100)"},
					},
				},
			},
		},
		Database: Database{Type: "mysql", Version: "8.0"},
	}

	resp, err := svc.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp.SQL == "" {
		t.Error("Expected non-empty SQL")
	}

	if resp.ConversationID == "" {
		t.Error("Expected non-empty conversation ID")
	}
}

func TestService_Generate_ContinueConversation(t *testing.T) {
	provider := &mockProvider{}
	validator := NewSQLValidator()
	store := NewMemoryContextStore()
	svc := NewServiceWithContextStore(provider, validator, 2, store)

	req1 := &GenerateRequest{
		Query: "查询所有用户",
		Schema: Schema{
			Tables: []Table{
				{
					Name: "users",
					Columns: []Column{
						{Name: "id", Type: "int"},
						{Name: "name", Type: "varchar(100)"},
					},
				},
			},
		},
		Database: Database{Type: "mysql", Version: "8.0"},
	}

	resp1, err := svc.Generate(context.Background(), req1)
	if err != nil {
		t.Fatalf("First Generate failed: %v", err)
	}

	req2 := &GenerateRequest{
		Query:          "查询年龄大于30的用户",
		ConversationID: resp1.ConversationID,
	}

	resp2, err := svc.Generate(context.Background(), req2)
	if err != nil {
		t.Fatalf("Second Generate failed: %v", err)
	}

	if resp2.ConversationID != resp1.ConversationID {
		t.Error("Conversation ID should remain the same")
	}
}

func TestService_SchemaEqual(t *testing.T) {
	svc := &Service{}

	s1 := Schema{
		Tables: []Table{
			{Name: "users", Columns: []Column{{Name: "id"}, {Name: "name"}}},
		},
	}

	s2 := Schema{
		Tables: []Table{
			{Name: "users", Columns: []Column{{Name: "id"}, {Name: "name"}}},
		},
	}

	if !svc.schemaEqual(s1, s2) {
		t.Error("Expected schemas to be equal")
	}

	s3 := Schema{
		Tables: []Table{
			{Name: "orders", Columns: []Column{{Name: "id"}}},
		},
	}

	if svc.schemaEqual(s1, s3) {
		t.Error("Expected schemas to be different")
	}
}
