package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"text2sql/internal/text2sql"
)

// Handler API 处理器
type Handler struct {
	text2sql    *text2sql.Service
	apiKeys     map[string]bool
	validate    *validator.Validate
	rateLimiter *RateLimiter
}

const maxRequestBodyBytes int64 = 1 << 20 // 1MB

// NewHandler 创建 Handler
func NewHandler(text2sql *text2sql.Service, apiKeys []string) *Handler {
	keyMap := make(map[string]bool)
	for _, key := range apiKeys {
		keyMap[key] = true
	}
	return &Handler{
		text2sql:    text2sql,
		apiKeys:     keyMap,
		validate:    validator.New(),
		rateLimiter: NewRateLimiter(10, time.Minute), // 每分钟10个请求
	}
}

// Routes 注册路由
func (h *Handler) Routes(r chi.Router) {
	r.Get("/api/v1/health", h.Health)
	r.With(h.authMiddleware, h.rateLimitMiddleware).Post("/api/v1/sql/generate", h.Generate)
}

// authMiddleware API Key 认证
func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractAPIKey(r)
		if key == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "缺少 API Key")
			return
		}
		if !h.apiKeys[key] {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "API Key 无效")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractAPIKey(r *http.Request) string {
	if k := r.Header.Get("Authorization"); k != "" {
		if strings.HasPrefix(k, "Bearer ") {
			return strings.TrimPrefix(k, "Bearer ")
		}
	}
	return r.Header.Get("X-API-Key")
}

// Health 健康检查
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks": map[string]interface{}{
			"service": map[string]string{
				"status": "ok",
			},
		},
	}
	json.NewEncoder(w).Encode(health)
}

// Generate 生成 SQL
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" && !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Content-Type 必须为 application/json")
		return
	}

	var req text2sql.GenerateRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "INVALID_REQUEST", "请求体过大")
			return
		}
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "请求体解析失败: "+err.Error())
		return
	}

	if err := h.validate.Struct(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "参数校验失败: "+ve.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "参数校验失败")
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "query 不能为空")
		return
	}
	// 新会话或 conversation_id 无效时，schema 和 database 必填
	if req.ConversationID == "" && (len(req.Schema.Tables) == 0 || req.Database.Type == "") {
		if len(req.Schema.Tables) == 0 {
			writeError(w, http.StatusBadRequest, "INVALID_SCHEMA", "新会话需提供 schema.tables")
			return
		}
		writeError(w, http.StatusBadRequest, "INVALID_DATABASE", "新会话需提供 database.type")
		return
	}

	resp, err := h.text2sql.Generate(r.Context(), &req)
	if err != nil {
		if errors.Is(err, text2sql.ErrSQLValidation) {
			writeError(w, http.StatusBadRequest, "SQL_VALIDATION_FAILED", err.Error())
			return
		}
		if errors.Is(err, text2sql.ErrConversationNotFound) {
			writeError(w, http.StatusNotFound, "CONVERSATION_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, text2sql.ErrSchemaMismatch) {
			writeError(w, http.StatusBadRequest, "SCHEMA_MISMATCH", err.Error())
			return
		}
		if errors.Is(err, text2sql.ErrDatabaseMismatch) {
			writeError(w, http.StatusBadRequest, "DATABASE_MISMATCH", err.Error())
			return
		}
		if errors.Is(err, text2sql.ErrSchemaRequired) {
			writeError(w, http.StatusBadRequest, "SCHEMA_REQUIRED", err.Error())
			return
		}
		if errors.Is(err, text2sql.ErrDatabaseRequired) {
			writeError(w, http.StatusBadRequest, "DATABASE_REQUIRED", err.Error())
			return
		}
		if errors.Is(err, text2sql.ErrLLMError) {
			writeError(w, http.StatusInternalServerError, "LLM_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INVALID_REQUEST", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Code: code, Message: message})
}
