package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/auditlog/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// AuditLogService: tra cứu nhật ký thao tác (ghi nhật ký đi qua AuditService).
type AuditLogService interface {
	List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
	Get(ctx context.Context, id uint) (*v1.AuditLogDetail, error)
	Options(ctx context.Context) (*v1.OptionsRes, error)
}

type auditLogService struct {
	repo repository.AuditLogRepository
}

func NewAuditLogService(repo repository.AuditLogRepository) AuditLogService {
	return &auditLogService{repo: repo}
}

func auditRowToDTO(r repository.AuditLogRow) *v1.AuditLog {
	name := r.ActorName
	if r.ActorID == nil {
		name = "Hệ thống"
	}
	return &v1.AuditLog{
		ID: r.ID, ActorID: r.ActorID, ActorName: name, ActorEmail: r.ActorEmail, Action: r.Action,
		Entity: r.Entity, EntityID: r.EntityID, IP: r.IP, CreatedAt: r.CreatedAt,
	}
}

func (s *auditLogService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	f := repository.AuditLogFilter{ActorID: req.ActorID, Entity: req.Entity, EntityID: req.EntityID, Action: req.Action}
	if req.From != "" {
		t, err := time.ParseInLocation(consts.DateLayout, req.From, reportLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Từ ngày không hợp lệ")
		}
		f.From = &t
	}
	if req.To != "" {
		t, err := time.ParseInLocation(consts.DateLayout, req.To, reportLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Đến ngày không hợp lệ")
		}
		t = t.AddDate(0, 0, 1)
		f.To = &t
	}
	q := req.ToQuery([]string{"a.created_at", "a.id"}, "a.created_at")
	rows, total, err := s.repo.List(ctx, f, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, auditRowToDTO), total, q), nil
}

func (s *auditLogService) Get(ctx context.Context, id uint) (*v1.AuditLogDetail, error) {
	row, err := s.repo.Get(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy nhật ký")
	}
	if err != nil {
		return nil, err
	}
	before, after := auditRaw(row.Before), auditRaw(row.After)
	return &v1.AuditLogDetail{
		AuditLog: *auditRowToDTO(*row), Before: before, After: after, Changes: auditDiff(before, after),
	}, nil
}

func (s *auditLogService) Options(ctx context.Context) (*v1.OptionsRes, error) {
	ents, err := s.repo.Entities(ctx)
	if err != nil {
		return nil, err
	}
	acts, err := s.repo.Actions(ctx)
	if err != nil {
		return nil, err
	}
	if ents == nil {
		ents = []string{}
	}
	if acts == nil {
		acts = []string{}
	}
	return &v1.OptionsRes{Entities: ents, Actions: acts}, nil
}

var auditNull = json.RawMessage("null")

func auditRaw(b []byte) json.RawMessage {
	if len(b) == 0 {
		return auditNull
	}
	return json.RawMessage(b)
}

// auditDiff so sánh trường cấp 1 của hai object JSON. Không phải object (hoặc null) → so sánh toàn bộ dưới field "".
func auditDiff(before, after json.RawMessage) []v1.FieldChange {
	var bm, am map[string]json.RawMessage
	bErr := json.Unmarshal(before, &bm)
	aErr := json.Unmarshal(after, &am)
	if bErr != nil || aErr != nil || (bm == nil && am == nil) {
		if auditEqual(before, after) {
			return []v1.FieldChange{}
		}
		if bm == nil && am == nil {
			return []v1.FieldChange{{Field: "", Before: before, After: after}}
		}
	}
	keys := map[string]bool{}
	for k := range bm {
		keys[k] = true
	}
	for k := range am {
		keys[k] = true
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	out := []v1.FieldChange{}
	for _, k := range names {
		b, okB := bm[k]
		a, okA := am[k]
		if !okB {
			b = auditNull
		}
		if !okA {
			a = auditNull
		}
		if !auditEqual(b, a) {
			out = append(out, v1.FieldChange{Field: k, Before: b, After: a})
		}
	}
	return out
}

// auditEqual so sánh hai giá trị JSON theo ngữ nghĩa (bỏ qua khoảng trắng/thứ tự khóa).
func auditEqual(a, b json.RawMessage) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return bytes.Equal(a, b)
	}
	xb, _ := json.Marshal(x)
	yb, _ := json.Marshal(y)
	return bytes.Equal(xb, yb)
}
