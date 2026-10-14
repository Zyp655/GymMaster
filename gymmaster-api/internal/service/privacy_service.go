package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/privacy/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/cryptoext"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityUserMFA     = "user_mfa"
	auditEntityDataRequest = "data_request"
	defaultMFAEncryptionKey = "gymmaster-default-mfa-encryption-key-32"
)

var (
	dataRequestOrderable = []string{"id", "type", "status", "created_at"}
	validConsentTypes    = []string{entity.ConsentTypeTerms, entity.ConsentTypeMarketing, entity.ConsentTypePhoto}
)

type (
	PrivacyService interface {
		GetMFAStatus(ctx context.Context) (*v1.MFAStatusRes, error)
		SetupMFA(ctx context.Context) (*v1.SetupMFARes, error)
		EnableMFA(ctx context.Context, req *v1.EnableMFAReq) (*v1.EnableMFARes, error)
		DisableMFA(ctx context.Context, req *v1.DisableMFAReq) (*v1.DisableMFARes, error)
		VerifyMFA(ctx context.Context, req *v1.VerifyMFAReq) (*v1.VerifyMFARes, error)

		ListSessions(ctx context.Context) (v1.ListSessionsRes, error)
		RevokeSession(ctx context.Context, sessionID uint) error
		RevokeOtherSessions(ctx context.Context, req *v1.RevokeOtherSessionsReq) (*v1.RevokeOtherSessionsRes, error)

		MemberConsents(ctx context.Context) (v1.MemberConsentsRes, error)
		UpdateMemberConsents(ctx context.Context, req *v1.UpdateMemberConsentsReq) (v1.UpdateMemberConsentsRes, error)
		MemberDataRequests(ctx context.Context) (v1.MemberDataRequestsRes, error)
		CreateDataRequest(ctx context.Context, req *v1.CreateDataRequestReq) (*v1.DataRequestItem, error)

		ListDataRequests(ctx context.Context, req *v1.ListDataRequestsReq) (*v1.ListDataRequestsRes, error)
		ProcessDataRequest(ctx context.Context, req *v1.ProcessDataRequestReq) (*v1.DataRequestItem, error)
	}

	privacyService struct {
		repo   repository.PrivacyRepository
		tx     postgres.TxManager
		audit  AuditService
		encKey string
		now    func() time.Time
	}
)

func NewPrivacyService(repo repository.PrivacyRepository, tx postgres.TxManager, audit AuditService, encKey string) PrivacyService {
	if strings.TrimSpace(encKey) == "" {
		encKey = defaultMFAEncryptionKey
	}
	return &privacyService{repo: repo, tx: tx, audit: audit, encKey: encKey, now: time.Now}
}

func (s *privacyService) runTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx != nil {
		return s.tx.WithTx(ctx, fn)
	}
	return fn(ctx)
}

func (s *privacyService) requireUserID(ctx context.Context) (uint, error) {
	uid := reqctx.UserID(ctx)
	if uid == nil || *uid == 0 {
		return 0, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	return *uid, nil
}

func (s *privacyService) currentMember(ctx context.Context) (*entity.Member, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	m, err := s.repo.FindMemberByUserID(ctx, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Tài khoản chưa được liên kết với hội viên")
		}
		return nil, err
	}
	return m, nil
}

// ---- MFA (2FA TOTP) -------------------------------------------------------------

func parseRecoveryHashes(raw datatypes.JSON) []string {
	var hashes []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &hashes)
	}
	return hashes
}

func generateRecoveryCodes(n int) ([]string, datatypes.JSON) {
	codes := make([]string, 0, n)
	hashes := make([]string, 0, n)
	for range n {
		var b [4]byte
		_, _ = rand.Read(b[:])
		hx := strings.ToUpper(hex.EncodeToString(b[:]))
		code := fmt.Sprintf("%s-%s", hx[:4], hx[4:])
		codes = append(codes, code)
		hashes = append(hashes, hash.SHA256(code))
	}
	b, _ := json.Marshal(hashes)
	return codes, datatypes.JSON(b)
}

func (s *privacyService) GetMFAStatus(ctx context.Context) (*v1.MFAStatusRes, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	mfa, err := s.repo.FindUserMFA(ctx, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &v1.MFAStatusRes{Enabled: false}, nil
		}
		return nil, err
	}
	hashes := parseRecoveryHashes(mfa.RecoveryCodesHash)
	return &v1.MFAStatusRes{
		Enabled:                mfa.EnabledAt != nil,
		SetupPending:           mfa.EnabledAt == nil && mfa.TOTPSecretEnc != "",
		EnabledAt:              mfa.EnabledAt,
		RecoveryCodesRemaining: len(hashes),
	}, nil
}

func (s *privacyService) SetupMFA(ctx context.Context) (*v1.SetupMFARes, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	email := fmt.Sprintf("user-%d@gymmaster.vn", uid)
	if u, err := s.repo.FindUserByID(ctx, uid); err == nil && u != nil && u.Email != "" {
		email = u.Email
	}
	secret, err := cryptoext.GenerateTOTPSecret()
	if err != nil {
		return nil, err
	}
	enc, err := cryptoext.EncryptAESGCM(secret, s.encKey)
	if err != nil {
		return nil, err
	}
	mfa := &entity.UserMFA{
		UserID:            uid,
		TOTPSecretEnc:     enc,
		EnabledAt:         nil,
		RecoveryCodesHash: datatypes.JSON([]byte("[]")),
	}
	if err := s.repo.SaveUserMFA(ctx, mfa); err != nil {
		return nil, err
	}
	return &v1.SetupMFARes{
		Secret:     secret,
		OTPAuthURL: cryptoext.BuildOTPAuthURL(email, secret),
	}, nil
}

func (s *privacyService) EnableMFA(ctx context.Context, req *v1.EnableMFAReq) (*v1.EnableMFARes, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	mfa, err := s.repo.FindUserMFA(ctx, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng khởi tạo 2FA trước khi kích hoạt")
		}
		return nil, err
	}
	secret, err := cryptoext.DecryptAESGCM(mfa.TOTPSecretEnc, s.encKey)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Khóa bí mật 2FA không hợp lệ, vui lòng thiết lập lại")
	}
	now := s.now()
	if !cryptoext.VerifyTOTPCode(secret, req.Code, now) {
		return nil, rescode.New(rescode.MFAInvalidCode, "Mã xác thực 2FA không hợp lệ")
	}
	codes, hashesJSON := generateRecoveryCodes(6)
	mfa.EnabledAt = &now
	mfa.RecoveryCodesHash = hashesJSON
	if err := s.repo.SaveUserMFA(ctx, mfa); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityUserMFA, uid, nil, map[string]any{"enabled": true})
	}
	return &v1.EnableMFARes{
		Enabled:       true,
		EnabledAt:     mfa.EnabledAt,
		RecoveryCodes: codes,
	}, nil
}

func (s *privacyService) verifyUserCodeOrRecovery(ctx context.Context, mfa *entity.UserMFA, code string) (bool, bool, error) {
	secret, err := cryptoext.DecryptAESGCM(mfa.TOTPSecretEnc, s.encKey)
	if err != nil {
		return false, false, err
	}
	trimmed := strings.ToUpper(strings.TrimSpace(code))
	if cryptoext.VerifyTOTPCode(secret, trimmed, s.now()) {
		return true, false, nil
	}
	// Check recovery codes
	hashes := parseRecoveryHashes(mfa.RecoveryCodesHash)
	targetHash := hash.SHA256(trimmed)
	idx := slices.Index(hashes, targetHash)
	if idx >= 0 {
		hashes = slices.Delete(hashes, idx, idx+1)
		b, _ := json.Marshal(hashes)
		mfa.RecoveryCodesHash = datatypes.JSON(b)
		if err := s.repo.SaveUserMFA(ctx, mfa); err != nil {
			return false, false, err
		}
		return true, true, nil
	}
	return false, false, nil
}

func (s *privacyService) DisableMFA(ctx context.Context, req *v1.DisableMFAReq) (*v1.DisableMFARes, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	mfa, err := s.repo.FindUserMFA(ctx, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &v1.DisableMFARes{Enabled: false}, nil
		}
		return nil, err
	}
	ok, _, err := s.verifyUserCodeOrRecovery(ctx, mfa, req.Code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, rescode.New(rescode.MFAInvalidCode, "Mã xác thực 2FA hoặc mã khôi phục không hợp lệ")
	}
	if err := s.repo.DeleteUserMFA(ctx, uid); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityUserMFA, uid, map[string]any{"enabled": true}, map[string]any{"enabled": false})
	}
	return &v1.DisableMFARes{Enabled: false}, nil
}

func (s *privacyService) VerifyMFA(ctx context.Context, req *v1.VerifyMFAReq) (*v1.VerifyMFARes, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	mfa, err := s.repo.FindUserMFA(ctx, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || mfa == nil || mfa.EnabledAt == nil {
			return nil, rescode.New(rescode.BadRequest, "Tài khoản chưa bật xác thực 2FA")
		}
		return nil, err
	}
	ok, usedRec, err := s.verifyUserCodeOrRecovery(ctx, mfa, req.Code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, rescode.New(rescode.MFAInvalidCode, "Mã xác thực 2FA không hợp lệ")
	}
	return &v1.VerifyMFARes{Verified: true, UsedRecovery: usedRec}, nil
}

// ---- Sessions -------------------------------------------------------------------

func (s *privacyService) ListSessions(ctx context.Context) (v1.ListSessionsRes, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListActiveSessions(ctx, uid, s.now())
	if err != nil {
		return nil, err
	}
	out := make([]*v1.SessionItem, 0, len(rows))
	for _, rt := range rows {
		devName := rt.DeviceName
		if devName == "" {
			devName = summarizeUserAgent(rt.UserAgent)
		}
		out = append(out, &v1.SessionItem{
			ID:         rt.ID,
			UserID:     rt.UserID,
			UserAgent:  rt.UserAgent,
			IP:         rt.IP,
			DeviceName: devName,
			LastUsedAt: rt.LastUsedAt,
			ExpiresAt:  rt.ExpiresAt,
			CreatedAt:  rt.CreatedAt,
		})
	}
	return out, nil
}

func summarizeUserAgent(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return "Thiết bị không xác định"
	}
	if len(ua) > 80 {
		return ua[:80]
	}
	return ua
}

func (s *privacyService) RevokeSession(ctx context.Context, sessionID uint) error {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return err
	}
	ok, err := s.repo.RevokeSession(ctx, uid, sessionID, s.now())
	if err != nil {
		return err
	}
	if !ok {
		return rescode.New(rescode.NotFound, "Không tìm thấy phiên đăng nhập")
	}
	return nil
}

func (s *privacyService) RevokeOtherSessions(ctx context.Context, req *v1.RevokeOtherSessionsReq) (*v1.RevokeOtherSessionsRes, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	tokenHash := ""
	if strings.TrimSpace(req.RefreshToken) != "" {
		tokenHash = hash.SHA256(strings.TrimSpace(req.RefreshToken))
	}
	cnt, err := s.repo.RevokeOtherSessions(ctx, uid, req.CurrentSessionID, tokenHash, s.now())
	if err != nil {
		return nil, err
	}
	return &v1.RevokeOtherSessionsRes{RevokedCount: cnt}, nil
}

// ---- Member Consents & Data Requests (Nghị định 13/2023) ------------------------

func (s *privacyService) MemberConsents(ctx context.Context) (v1.MemberConsentsRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListMemberConsents(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	byType := map[string]entity.Consent{}
	for _, c := range rows {
		byType[c.Type] = c
	}
	now := s.now()
	out := make([]*v1.ConsentItem, 0, len(validConsentTypes))
	for _, ct := range validConsentTypes {
		if existing, ok := byType[ct]; ok {
			out = append(out, &v1.ConsentItem{
				Type:      existing.Type,
				Granted:   existing.Granted,
				GrantedAt: existing.GrantedAt,
				IP:        existing.IP,
			})
		} else {
			out = append(out, &v1.ConsentItem{
				Type:      ct,
				Granted:   ct == entity.ConsentTypeTerms,
				GrantedAt: now,
			})
		}
	}
	return out, nil
}

func (s *privacyService) UpdateMemberConsents(ctx context.Context, req *v1.UpdateMemberConsentsReq) (v1.UpdateMemberConsentsRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	ip := ""
	if r := g.RequestFromCtx(ctx); r != nil {
		ip = r.GetClientIp()
	}
	now := s.now()
	updates := map[string]bool{}
	if req.Terms != nil {
		updates[entity.ConsentTypeTerms] = *req.Terms
	}
	if req.Marketing != nil {
		updates[entity.ConsentTypeMarketing] = *req.Marketing
	}
	if req.Photo != nil {
		updates[entity.ConsentTypePhoto] = *req.Photo
	}
	for _, it := range append(req.Consents, req.Items...) {
		ct := strings.ToUpper(strings.TrimSpace(it.Type))
		if slices.Contains(validConsentTypes, ct) {
			updates[ct] = it.Granted
		}
	}
	for ct, granted := range updates {
		c := &entity.Consent{
			MemberID:  m.ID,
			Type:      ct,
			Granted:   granted,
			GrantedAt: now,
			IP:        ip,
		}
		if err := s.repo.UpsertConsent(ctx, c); err != nil {
			return nil, err
		}
	}
	return s.MemberConsents(ctx)
}

func (s *privacyService) MemberDataRequests(ctx context.Context) (v1.MemberDataRequestsRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListMemberDataRequests(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, func(r repository.DataRequestRow) *v1.DataRequestItem {
		return toDataRequestItem(r, nil)
	}), nil
}

func (s *privacyService) CreateDataRequest(ctx context.Context, req *v1.CreateDataRequestReq) (*v1.DataRequestItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	reqType := strings.ToUpper(strings.TrimSpace(req.Type))
	if reqType != entity.DataRequestExport && reqType != entity.DataRequestDelete {
		return nil, rescode.New(rescode.BadRequest, "Loại yêu cầu phải là EXPORT hoặc DELETE")
	}
	now := s.now()
	dr := &entity.DataRequest{
		MemberID: m.ID,
		Type:     reqType,
		Status:   entity.DataRequestPending,
		Reason:   strings.TrimSpace(req.Reason),
	}
	var exportData map[string]any
	if reqType == entity.DataRequestExport {
		dr.Status = entity.DataRequestDone
		dr.DoneAt = &now
		dr.Note = "Hệ thống tự động trích xuất hồ sơ cá nhân theo Nghị định 13/2023/NĐ-CP"
		exportData = map[string]any{
			"member_id":     m.ID,
			"member_code":   m.MemberCode,
			"full_name":     m.FullName,
			"gender":        m.Gender,
			"phone":         m.Phone,
			"email":         m.Email,
			"id_card":       m.IDCard,
			"address":       m.Address,
			"loyalty_points": m.LoyaltyPoints,
			"exported_at":   now.Format(time.RFC3339),
		}
	}
	if err := s.repo.CreateDataRequest(ctx, dr); err != nil {
		return nil, err
	}
	row := repository.DataRequestRow{
		DataRequest: *dr,
		MemberCode:  m.MemberCode,
		MemberName:  m.FullName,
		MemberPhone: m.Phone,
	}
	return toDataRequestItem(row, exportData), nil
}

// ---- Admin: Process Data Requests -----------------------------------------------

func (s *privacyService) ListDataRequests(ctx context.Context, req *v1.ListDataRequestsReq) (*v1.ListDataRequestsRes, error) {
	q := req.PageReq.ToQuery(dataRequestOrderable, "id")
	q.OrderBy = "dr." + q.OrderBy
	rows, total, err := s.repo.ListDataRequests(ctx, q, strings.TrimSpace(req.Status), strings.TrimSpace(req.Type))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.DataRequestRow) *v1.DataRequestItem {
		return toDataRequestItem(r, nil)
	}), total, q), nil
}

func (s *privacyService) ProcessDataRequest(ctx context.Context, req *v1.ProcessDataRequestReq) (*v1.DataRequestItem, error) {
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if action != "APPROVE" && action != "REJECT" {
		return nil, rescode.New(rescode.BadRequest, "Hành động xử lý phải là APPROVE hoặc REJECT")
	}
	row, err := s.repo.FindDataRequestByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy yêu cầu dữ liệu cá nhân")
		}
		return nil, err
	}
	if row.Status == entity.DataRequestDone || row.Status == entity.DataRequestRejected {
		return nil, rescode.New(rescode.Conflict, "Yêu cầu này đã được xử lý trước đó")
	}

	now := s.now()
	uid := reqctx.UserID(ctx)
	dr := row.DataRequest
	dr.HandledBy = uid
	dr.Note = strings.TrimSpace(req.Note)
	dr.DoneAt = &now

	err = s.runTx(ctx, func(txCtx context.Context) error {
		if action == "APPROVE" {
			dr.Status = entity.DataRequestDone
			if dr.Type == entity.DataRequestDelete {
				if err := s.repo.AnonymizeMember(txCtx, dr.MemberID, uid); err != nil {
					return err
				}
			}
		} else {
			dr.Status = entity.DataRequestRejected
		}
		return s.repo.UpdateDataRequest(txCtx, &dr)
	})
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.FindDataRequestByID(ctx, req.ID)
	if err != nil {
		row.DataRequest = dr
		updated = row
	}
	res := toDataRequestItem(*updated, nil)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityDataRequest, dr.ID, row.Status, res)
	}
	return res, nil
}

func toDataRequestItem(r repository.DataRequestRow, exportData map[string]any) *v1.DataRequestItem {
	return &v1.DataRequestItem{
		ID:          r.ID,
		MemberID:    r.MemberID,
		MemberCode:  r.MemberCode,
		MemberName:  r.MemberName,
		MemberPhone: r.MemberPhone,
		Type:        r.Type,
		Status:      r.Status,
		Reason:      r.Reason,
		FileURL:     r.FileURL,
		HandledBy:   r.HandledBy,
		HandlerName: r.HandlerName,
		Note:        r.Note,
		ExportData:  exportData,
		CreatedAt:   r.CreatedAt,
		DoneAt:      r.DoneAt,
	}
}
