package helper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, gin.H{"status": "ok"})

	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Success)
	assert.Equal(t, "ok", body.Data["status"])
}

func TestSuccessAlwaysStatus200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Success hardcodes 200 even if a prior status was written.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, []int{1, 2, 3})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"success":true,"data":[1,2,3]}`, w.Body.String())
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		code int
		msg  string
	}{
		{"bad request", http.StatusBadRequest, "invalid type"},
		{"not found", http.StatusNotFound, "cafe not found"},
		{"internal", http.StatusInternalServerError, "search failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			Error(c, tt.code, tt.msg)

			assert.Equal(t, tt.code, w.Code)

			var body struct {
				Success bool   `json:"success"`
				Error   string `json:"error"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.False(t, body.Success)
			assert.Equal(t, tt.msg, body.Error)
		})
	}
}
