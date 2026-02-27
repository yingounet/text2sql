package text2sql

import "errors"

var (
	ErrSQLValidation        = errors.New("SQL_VALIDATION_FAILED")
	ErrConversationNotFound = errors.New("CONVERSATION_NOT_FOUND")
	ErrSchemaMismatch       = errors.New("SCHEMA_MISMATCH")
	ErrDatabaseMismatch     = errors.New("DATABASE_MISMATCH")
	ErrSchemaRequired       = errors.New("SCHEMA_REQUIRED")
	ErrDatabaseRequired     = errors.New("DATABASE_REQUIRED")
	ErrLLMError             = errors.New("LLM_ERROR")
)
