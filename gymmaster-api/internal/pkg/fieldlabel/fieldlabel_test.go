package fieldlabel

import "testing"

func TestToSnakeAndLabel(t *testing.T) {
	for in, want := range map[string]string{"BranchID": "branch_id", "IDCard": "id_card", "BodyHTML": "body_html",
		"ID": "id", "TransferFee": "transfer_fee", "SKU": "sku", "branch_id": "branch_id", "DurationMin": "duration_min"} {
		if got := ToSnake(in); got != want {
			t.Errorf("ToSnake(%q)=%q want %q", in, got, want)
		}
	}
	if l, ok := Label("BranchID"); !ok || l != "chi nhánh" {
		t.Errorf("Label(BranchID)=%q", l)
	}
	if Capitalize("chi nhánh") != "Chi nhánh" || Capitalize("ảnh") != "Ảnh" {
		t.Error("Capitalize")
	}
}
