package token

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("token không hợp lệ")
	ErrExpiredToken = errors.New("token đã hết hạn")
)

// Payload là thông tin người dùng nhúng trong access token.
// Quyền được nhúng sẵn để middleware không phải truy vấn DB ở mỗi request;
// thay đổi quyền có hiệu lực ở lần refresh kế tiếp (tối đa = AccessTTL).
type Payload struct {
	UserID      uint     `json:"uid"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"perms"`
	BranchIDs   []uint   `json:"branches"`
	// SessionID = id refresh token (phiên đăng nhập) cấp cùng access token này; 0 với token cũ.
	// Dùng để nhận biết "phiên hiện tại" (danh sách phiên, đăng xuất thiết bị khác).
	SessionID uint `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

func (p *Payload) HasRole(role string) bool {
	return slices.Contains(p.Roles, role)
}

func (p *Payload) HasPermission(perm string) bool {
	return slices.Contains(p.Permissions, perm)
}

// Maker tạo và kiểm tra access token.
type Maker interface {
	CreateToken(userID uint, roles, permissions []string, branchIDs []uint) (string, *Payload, error)
	// CreateSessionToken giống CreateToken nhưng gắn id phiên (refresh token) vào claim "sid".
	CreateSessionToken(userID, sessionID uint, roles, permissions []string, branchIDs []uint) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}

type jwtMaker struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

const minSecretSize = 32

func NewJWTMaker(secret, issuer string, ttl time.Duration) (Maker, error) {
	if len(secret) < minSecretSize {
		return nil, fmt.Errorf("JWT_SECRET phải có ít nhất %d ký tự", minSecretSize)
	}
	return &jwtMaker{secret: []byte(secret), issuer: issuer, ttl: ttl}, nil
}

func (m *jwtMaker) CreateToken(userID uint, roles, permissions []string, branchIDs []uint) (string, *Payload, error) {
	return m.CreateSessionToken(userID, 0, roles, permissions, branchIDs)
}

func (m *jwtMaker) CreateSessionToken(userID, sessionID uint, roles, permissions []string, branchIDs []uint) (string, *Payload, error) {
	now := time.Now()
	p := &Payload{
		UserID:      userID,
		SessionID:   sessionID,
		Roles:       roles,
		Permissions: permissions,
		BranchIDs:   branchIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, p).SignedString(m.secret)
	if err != nil {
		return "", nil, err
	}
	return s, p, nil
}

func (m *jwtMaker) VerifyToken(tokenStr string) (*Payload, error) {
	p := &Payload{}
	_, err := jwt.ParseWithClaims(tokenStr, p, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	return p, nil
}
