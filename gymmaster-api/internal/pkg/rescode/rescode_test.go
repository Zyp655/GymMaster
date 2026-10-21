package rescode

import "testing"

func TestHTTPStatus(t *testing.T) {
	cases := map[int]int{
		40000: 400, 40106: 401, 40108: 401, 40301: 403, 40400: 404, 40904: 409, 42900: 429,
		42001: 403, 42004: 403, 43005: 409, 43006: 409, 44001: 400, 44003: 400,
		46001: 409, 46002: 409, 47001: 400, 47002: 409, 47003: 409, 50000: 500,
	}
	for code, want := range cases {
		if got, ok := HTTPStatus(code); !ok || got != want {
			t.Errorf("HTTPStatus(%d) = %d,%v want %d", code, got, ok, want)
		}
	}
	for _, c := range []int{0, 51, 68, 9999, 60000} {
		if _, ok := HTTPStatus(c); ok {
			t.Errorf("HTTPStatus(%d) must not be a business code", c)
		}
	}
}
