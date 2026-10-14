package mailer

import (
	"context"
	"strings"
	"testing"
)

func TestRenderTemplates(t *testing.T) {
	html, err := Render(TplSubscriptionExpiring, ExpiringData{
		MemberName: "Nguyễn <Lan>", PlanName: "Gói 3 tháng", EndDate: "17/10/2026", DaysLeft: 3, BranchName: "Cầu Giấy",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Gói tập sắp hết hạn", "Nguyễn &lt;Lan&gt;", "17/10/2026", "còn 3 ngày", "Cầu Giấy", "GymMaster"} {
		if !strings.Contains(html, want) {
			t.Errorf("thiếu %q trong email", want)
		}
	}
	html, _ = Render(TplSubscriptionExpiring, ExpiringData{MemberName: "A", PlanName: "P", EndDate: "x"})
	if !strings.Contains(html, "(hôm nay)") {
		t.Error("DaysLeft = 0 phải hiển thị 'hôm nay'")
	}
	if _, err := Render(TplSubscriptionExpired, ExpiredData{MemberName: "A", PlanName: "P", EndDate: "x"}); err != nil {
		t.Fatal(err)
	}
	html, _ = Render(TplGeneric, GenericData{Title: "Tiêu đề", Paragraphs: []string{"Đoạn 1", "Đoạn 2"}})
	if !strings.Contains(html, "<title>Tiêu đề</title>") || !strings.Contains(html, "Đoạn 2") {
		t.Error("generic template sai")
	}
	if _, err := Render("khong-co", nil); err == nil {
		t.Error("template không tồn tại phải lỗi")
	}
}

func TestSendRequiresRecipient(t *testing.T) {
	m := New(MailCfgForTest())
	if err := m.Send(context.Background(), Message{To: []string{" "}, Subject: "x"}); err != ErrNoRecipient {
		t.Fatalf("err = %v", err)
	}
}
