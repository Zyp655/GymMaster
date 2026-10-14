package consts

// Mã quyền (permission code) dạng <module>.<action>, khớp bảng permissions (migrations 000001–000010).
// Dùng trong tag g.Meta: perm:"member.view" (nhiều quyền ngăn bởi dấu phẩy = chỉ cần một).
const (
	PermBranchView   = "branch.view"
	PermBranchCreate = "branch.create"
	PermBranchUpdate = "branch.update"
	PermBranchDelete = "branch.delete"

	PermRoleView   = "role.view"
	PermRoleCreate = "role.create"
	PermRoleUpdate = "role.update"
	PermRoleDelete = "role.delete"

	PermStaffView   = "staff.view"
	PermStaffCreate = "staff.create"
	PermStaffUpdate = "staff.update"
	PermStaffDelete = "staff.delete"

	PermRoomView   = "room.view"
	PermRoomManage = "room.manage"

	PermShiftView   = "shift.view"
	PermShiftManage = "shift.manage"

	PermMemberView   = "member.view"
	PermMemberCreate = "member.create"
	PermMemberUpdate = "member.update"
	PermMemberDelete = "member.delete"
	PermMemberMetric = "member.metric"
	PermMemberImport = "member.import"

	PermPlanView   = "plan.view"
	PermPlanManage = "plan.manage"

	PermPromotionView   = "promotion.view"
	PermPromotionManage = "promotion.manage"

	PermSettingManage = "setting.manage"

	PermSubscriptionView          = "subscription.view"
	PermSubscriptionSell          = "subscription.sell"
	PermSubscriptionFreeze        = "subscription.freeze"
	PermSubscriptionApproveFreeze = "subscription.approve_freeze"
	PermSubscriptionTransfer      = "subscription.transfer"
	PermSubscriptionCancel        = "subscription.cancel"

	PermInvoiceView    = "invoice.view"
	PermInvoiceCollect = "invoice.collect"
	PermInvoiceVoid    = "invoice.void"
	PermInvoiceRefund  = "invoice.refund"

	PermCheckinView = "checkin.view"
	PermCheckinScan = "checkin.scan"

	PermClassView   = "class.view"
	PermClassManage = "class.manage"

	PermBookingManage = "booking.manage"

	PermPtView           = "pt.view"
	PermPtSell           = "pt.sell"
	PermPtSchedule       = "pt.schedule"
	PermPtComplete       = "pt.complete"
	PermPtCancelContract = "pt.cancel_contract"

	PermCommissionView = "commission.view"
	PermCommissionPay  = "commission.pay"

	PermProductView   = "product.view"
	PermProductManage = "product.manage"

	PermInventoryView     = "inventory.view"
	PermInventoryImport   = "inventory.import"
	PermInventoryTransfer = "inventory.transfer"
	PermInventoryAdjust   = "inventory.adjust"

	PermPosSell = "pos.sell"

	PermReportView   = "report.view"
	PermReportExport = "report.export"

	PermAuditView = "audit.view"

	PermJobView = "job.view"
)
