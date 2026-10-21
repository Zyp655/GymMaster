package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
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
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/clientip"
	"gymmaster-api/internal/pkg/cryptoext"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityUserMFA      = "user_mfa"
	auditEntityDataRequest  = "data_request"
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
		// CheckLoginMFA dùng ở bước đăng nhập (cài đặt LoginMFAChecker cho AuthService).
		CheckLoginMFA(ctx context.Context, userID uint, code string) (enabled bool, ok bool, err error)

		ListSessions(ctx context.Context) (v1.ListSessionsRes, error)
		RevokeSession(ctx context.Context, sessionID uint) error
		RevokeOtherSessions(ctx context.Context, req *v1.RevokeOtherSessionsReq) (*v1.RevokeOtherSessionsRes, error)

		MemberConsents(ctx context.Context) (v1.MemberConsentsRes, error)
		UpdateMemberConsents(ctx context.Context, req *v1.UpdateMemberConsentsReq) (v1.UpdateMemberConsentsRes, error)
		MemberDataRequests(ctx context.Context) (v1.MemberDataRequestsRes, error)
		CreateDataRequest(ctx context.Context, req *v1.CreateDataRequestReq) (*v1.DataRequestItem, error)
		// DownloadDataRequest trả bản trích xuất JSON của yêu cầu EXPORT đã xong (chỉ chủ yêu cầu; khác → 404).
		DownloadDataRequest(ctx context.Context, id uint) ([]byte, string, error)

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
	// Đã bật 2FA thì không cho khởi tạo lại: nếu ghi đè, kẻ chiếm được phiên đăng nhập có thể
	// vô hiệu 2FA (EnabledAt = nil) mà không cần mã. Muốn đổi thiết bị phải tắt 2FA bằng mã hợp lệ trước.
	if existing, err := s.repo.FindUserMFA(ctx, uid); err == nil && existing != nil && existing.EnabledAt != nil {
		return nil, rescode.New(rescode.Conflict, "Xác thực 2 lớp đang bật. Vui lòng tắt 2FA bằng mã xác thực hợp lệ trước khi thiết lập lại")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
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
	if mfa.EnabledAt != nil {
		return nil, rescode.New(rescode.Conflict, "Xác thực 2 lớp đã được bật trước đó")
	}
	secret, err := cryptoext.DecryptAESGCM(mfa.TOTPSecretEnc, s.encKey)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Khóa bí mật 2FA không hợp lệ, vui lòng thiết lập lại")
	}
	now := s.now()
	step, ok := matchTOTPStep(secret, strings.TrimSpace(req.Code), now)
	if !ok {
		return nil, rescode.New(rescode.MFAInvalidCode, "Mã xác thực 2FA không hợp lệ")
	}
	codes, hashesJSON := generateRecoveryCodes(6)
	mfa.EnabledAt = &now
	mfa.RecoveryCodesHash = hashesJSON
	err = s.runTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SaveUserMFA(ctx, mfa); err != nil {
			return err
		}
		// Mã vừa dùng để kích hoạt không được dùng lại để đăng nhập/tắt 2FA.
		_, err := s.repo.ConsumeTOTPStep(ctx, uid, step)
		return err
	})
	if err != nil {
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

// matchTOTPStep trả bước thời gian (unix/30) mà mã TOTP khớp, cho phép lệch ±1 bước như VerifyTOTPCode.
func matchTOTPStep(secret, code string, now time.Time) (int64, bool) {
	if len(code) != 6 {
		return 0, false
	}
	base := now.Unix() / totpStepSeconds
	for offset := int64(-1); offset <= 1; offset++ {
		step := base + offset
		expected, err := cryptoext.GenerateTOTPCode(secret, time.Unix(step*totpStepSeconds, 0))
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

const totpStepSeconds = 30

// normalizeRecoveryCode: mã khôi phục dạng "ABCD-1234"; chấp nhận chữ thường và thiếu dấu gạch (gõ trên điện thoại).
func normalizeRecoveryCode(code string) string {
	c := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	if len(c) == 8 && !strings.Contains(c, "-") {
		c = c[:4] + "-" + c[4:]
	}
	return c
}

// verifyUserCodeOrRecovery kiểm tra mã TOTP hoặc mã khôi phục và TIÊU THỤ nó một cách nguyên tử ở DB:
//   - TOTP: bước thời gian phải mới hơn bước đã dùng gần nhất (chống phát lại mã trong cửa sổ ±30 giây);
//   - mã khôi phục: xóa khỏi danh sách bằng UPDATE có điều kiện (hai request song song chỉ một request thắng).
func (s *privacyService) verifyUserCodeOrRecovery(ctx context.Context, mfa *entity.UserMFA, code string) (bool, bool, error) {
	secret, err := cryptoext.DecryptAESGCM(mfa.TOTPSecretEnc, s.encKey)
	if err != nil {
		return false, false, err
	}
	trimmed := strings.TrimSpace(code)
	if step, ok := matchTOTPStep(secret, trimmed, s.now()); ok {
		fresh, err := s.repo.ConsumeTOTPStep(ctx, mfa.UserID, step)
		if err != nil {
			return false, false, err
		}
		return fresh, false, nil
	}
	recovery := normalizeRecoveryCode(trimmed)
	if !slices.Contains(parseRecoveryHashes(mfa.RecoveryCodesHash), hash.SHA256(recovery)) {
		return false, false, nil
	}
	consumed, err := s.repo.ConsumeRecoveryCode(ctx, mfa.UserID, hash.SHA256(recovery))
	if err != nil {
		return false, false, err
	}
	return consumed, consumed, nil
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

// CheckLoginMFA: enabled=false nếu user chưa bật 2FA; ngược lại ok cho biết mã TOTP/mã khôi phục có hợp lệ.
// code rỗng chỉ trả trạng thái (không tốn mã khôi phục).
func (s *privacyService) CheckLoginMFA(ctx context.Context, userID uint, code string) (bool, bool, error) {
	mfa, err := s.repo.FindUserMFA(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, nil
		}
		return false, false, err
	}
	if mfa.EnabledAt == nil {
		return false, false, nil
	}
	if strings.TrimSpace(code) == "" {
		return true, false, nil
	}
	ok, _, err := s.verifyUserCodeOrRecovery(ctx, mfa, code)
	if err != nil {
		return true, false, err
	}
	return true, ok, nil
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
	currentID := currentSessionID(ctx)
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
			IsCurrent:  currentID > 0 && rt.ID == currentID,
		})
	}
	return out, nil
}

func summarizeUserAgent(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return "Thiết bị không xác định"
	}
	return truncate(ua, 80)
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

// currentSessionID: id phiên (refresh token) gắn trong access token đang gọi API (0 = token cũ không có sid).
func currentSessionID(ctx context.Context) uint {
	if p := reqctx.Auth(ctx); p != nil {
		return p.SessionID
	}
	return 0
}

// RevokeOtherSessions đăng xuất mọi thiết bị KHÁC: phiên hiện tại lấy từ access token (sid), hoặc từ
// current_session_id/refresh_token trong body. Không xác định được phiên hiện tại → 400 thay vì thu hồi
// luôn cả phiên đang dùng.
func (s *privacyService) RevokeOtherSessions(ctx context.Context, req *v1.RevokeOtherSessionsReq) (*v1.RevokeOtherSessionsRes, error) {
	uid, err := s.requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	keepID := currentSessionID(ctx)
	if keepID == 0 {
		keepID = req.CurrentSessionID
	}
	tokenHash := ""
	if strings.TrimSpace(req.RefreshToken) != "" {
		tokenHash = hash.SHA256(strings.TrimSpace(req.RefreshToken))
	}
	if keepID == 0 && tokenHash == "" {
		return nil, rescode.New(rescode.BadRequest, "Không xác định được phiên hiện tại, vui lòng đăng nhập lại rồi thử lại")
	}
	cnt, err := s.repo.RevokeOtherSessions(ctx, uid, keepID, tokenHash, s.now())
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
		ip = clientip.Real(r)
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
	pending, err := s.repo.HasPendingDataRequest(ctx, m.ID, reqType)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, rescode.New(rescode.Conflict, "Bạn đã có một yêu cầu cùng loại đang chờ xử lý")
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
		if exportData, err = s.buildMemberExport(ctx, m, now); err != nil {
			return nil, err
		}
	}
	err = s.runTx(ctx, func(ctx context.Context) error {
		if err := s.repo.CreateDataRequest(ctx, dr); err != nil {
			return mapDuplicate(err, "Bạn đã có một yêu cầu cùng loại đang chờ xử lý")
		}
		if exportData == nil {
			return nil
		}
		// Lưu bản trích xuất để hội viên tải lại bất cứ lúc nào qua file_url.
		b, err := json.Marshal(exportData)
		if err != nil {
			return err
		}
		dr.FileURL = dataRequestDownloadURL(dr.ID)
		if err := s.repo.SaveDataRequestExport(ctx, dr.ID, b); err != nil {
			return err
		}
		return s.repo.UpdateDataRequest(ctx, dr)
	})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityDataRequest, dr.ID, nil, map[string]any{"type": dr.Type, "status": dr.Status})
	}
	row := repository.DataRequestRow{
		DataRequest: *dr,
		MemberCode:  m.MemberCode,
		MemberName:  m.FullName,
		MemberPhone: m.Phone,
	}
	return toDataRequestItem(row, exportData), nil
}

// buildMemberExport gom dữ liệu cá nhân của hội viên để trích xuất (Nghị định 13/2023).
func (s *privacyService) buildMemberExport(ctx context.Context, m *entity.Member, now time.Time) (map[string]any, error) {
	consents, err := s.repo.ListMemberConsents(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	consentOut := make([]map[string]any, 0, len(consents))
	for _, c := range consents {
		consentOut = append(consentOut, map[string]any{"type": c.Type, "granted": c.Granted, "granted_at": c.GrantedAt})
	}
	var dob any
	if m.Dob != nil {
		dob = m.Dob.Format(consts.DateLayout)
	}
	return map[string]any{
		"member_id":      m.ID,
		"member_code":    m.MemberCode,
		"full_name":      m.FullName,
		"gender":         m.Gender,
		"dob":            dob,
		"phone":          m.Phone,
		"email":          m.Email,
		"id_card":        m.IDCard,
		"address":        m.Address,
		"loyalty_points": m.LoyaltyPoints,
		"consents":       consentOut,
		"exported_at":    now.Format(time.RFC3339),
	}, nil
}

// dataRequestDownloadURL: đường dẫn (tương đối với gốc API) tải bản trích xuất – xem DownloadDataRequestReq.
func dataRequestDownloadURL(id uint) string {
	return fmt.Sprintf("/v1/member/data-requests/%d/download", id)
}

func (s *privacyService) DownloadDataRequest(ctx context.Context, id uint) ([]byte, string, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, "", err
	}
	notFound := rescode.New(rescode.NotFound, "Không tìm thấy bản trích xuất dữ liệu")
	row, err := s.repo.FindDataRequestByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", notFound
		}
		return nil, "", err
	}
	// Không lộ yêu cầu của người khác: cùng trả 404.
	if row.MemberID != m.ID || row.Type != entity.DataRequestExport || row.Status != entity.DataRequestDone {
		return nil, "", notFound
	}
	data, err := s.repo.FindDataRequestExport(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Yêu cầu EXPORT cũ (trước khi lưu bản trích xuất): tạo lại từ hồ sơ hiện tại và lưu để lần sau dùng lại.
		exp, buildErr := s.buildMemberExport(ctx, m, s.now())
		if buildErr != nil {
			return nil, "", buildErr
		}
		if data, err = json.Marshal(exp); err != nil {
			return nil, "", err
		}
		err = s.repo.SaveDataRequestExport(ctx, id, data)
	}
	if err != nil {
		return nil, "", err
	}
	return data, fmt.Sprintf("du-lieu-ca-nhan-%s-%d.json", m.MemberCode, id), nil
}

// ---- Admin: Process Data Requests -----------------------------------------------

func (s *privacyService) ListDataRequests(ctx context.Context, req *v1.ListDataRequestsReq) (*v1.ListDataRequestsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem yêu cầu dữ liệu của chi nhánh này")
	}
	q := req.PageReq.ToQuery(dataRequestOrderable, "id")
	q.OrderBy = "dr." + q.OrderBy
	rows, total, err := s.repo.ListDataRequests(ctx, q, strings.TrimSpace(req.Status), strings.TrimSpace(req.Type), branchIDs)
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
	if !reqctx.CanAccessBranch(ctx, row.MemberBranchID) {
		// Không lộ sự tồn tại của yêu cầu thuộc chi nhánh khác.
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy yêu cầu dữ liệu cá nhân")
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
	if r.Type == entity.DataRequestExport && r.Status == entity.DataRequestDone && r.FileURL == "" {
		r.FileURL = dataRequestDownloadURL(r.ID)
	}
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
