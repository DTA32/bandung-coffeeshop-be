package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	c, w := newTestContext(http.MethodGet, "", nil, nil)

	Health(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"success":true,"data":{"status":"ok"}}`, w.Body.String())
}
