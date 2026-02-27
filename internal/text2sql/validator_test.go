package text2sql

import "testing"

func TestValidateMySQLReadOnly(t *testing.T) {
	v := NewSQLValidator()
	if err := v.Validate("SELECT * FROM users", "mysql", ""); err != nil {
		t.Fatalf("expected select to pass, got error: %v", err)
	}
}

func TestValidateRejectsDeleteWithComments(t *testing.T) {
	v := NewSQLValidator()
	sql := "DE/**/LETE FROM users"
	if err := v.Validate(sql, "mysql", ""); err == nil {
		t.Fatalf("expected non-readonly SQL to be rejected")
	}
}

func TestValidateRejectsMultipleStatements(t *testing.T) {
	v := NewSQLValidator()
	sql := "SELECT * FROM users; DELETE FROM users"
	if err := v.Validate(sql, "postgres", ""); err == nil {
		t.Fatalf("expected multiple statements to be rejected")
	}
}
