package middleware

import (
	"net/http"
	"testing"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/stretchr/testify/assert"

	"gymmaster-api/internal/pkg/rescode"
)

func TestHTTPStatus(t *testing.T) {
	cases := map[gcode.Code]int{
		gcode.CodeOK:               http.StatusOK,
		rescode.TokenExpired:       http.StatusUnauthorized,
		rescode.PermissionDenied:   http.StatusForbidden,
		rescode.NotFound:           http.StatusNotFound,
		rescode.Conflict:           http.StatusConflict,
		gcode.CodeValidationFailed: http.StatusBadRequest,
		gcode.CodeInternalError:    http.StatusInternalServerError,
		gcode.CodeDbOperationError: http.StatusInternalServerError,
	}
	for code, want := range cases {
		assert.Equal(t, want, httpStatus(code), code.Message())
	}
}
