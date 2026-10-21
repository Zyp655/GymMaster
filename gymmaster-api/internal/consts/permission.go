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

	PermJobView   = "job.view"
	PermJobManage = "job.manage" // chạy job thủ công, gửi email thử

	PermWorkoutView   = "workout.view"
	PermWorkoutManage = "workout.manage"

	PermReviewView = "review.view"

	PermEquipmentView   = "equipment.view"
	PermEquipmentManage = "equipment.manage"

	PermLeadView   = "lead.view"
	PermLeadManage = "lead.manage"

	// Giai đoạn 3 (Sprint 18–23)
	PermDeviceView   = "device.view"
	PermDeviceManage = "device.manage"

	PermLockerView   = "locker.view"
	PermLockerManage = "locker.manage"

	PermPayrollView   = "payroll.view"
	PermPayrollManage = "payroll.manage"

	PermExpenseView       = "expense.view"
	PermExpenseManage     = "expense.manage"
	PermCashSessionManage = "cash_session.manage"

	PermLoyaltyView   = "loyalty.view"
	PermLoyaltyManage = "loyalty.manage"

	PermEngagementView   = "engagement.view"
	PermEngagementManage = "engagement.manage"

	// Giai đoạn 4 (Sprint 24–27)
	PermContractManage      = "contract.manage"
	PermDayPassSell         = "day_pass.sell"
	PermLeaveManage         = "leave.manage"
	PermAvailabilityManage  = "availability.manage"
	PermLeaveRequest        = "leave.request"
	PermLeaveApprove        = "leave.approve"
	PermHolidayManage       = "holiday.manage"
	PermKpiManage           = "kpi.manage"
	PermReportCohort        = "report.cohort"
	PermPrivacyManage       = "privacy.manage"
	PermMemberViewSensitive = "member.view_sensitive"
	PermDataRequestProcess  = "data_request.process"

	// Bổ sung chi tiết phân quyền theo §16.3 & Sprint 29–30
	PermEquipmentReport    = "equipment.report"
	PermLeadConvert        = "lead.convert"
	PermAttendanceView     = "attendance.view"
	PermPayrollLock        = "payroll.lock"
	PermExpenseApprove     = "expense.approve"
	PermInstallmentManage  = "installment.manage"
	PermReportProfitLoss   = "report.profit_loss"
	PermLoyaltyAdjust      = "loyalty.adjust"
	PermNutritionManage    = "nutrition.manage"
	PermAnnouncementManage = "announcement.manage"
	PermSurveyManage       = "survey.manage"
	PermCmsManage          = "cms.manage"
	PermImportLegacy       = "import.legacy"
	PermSetupWizard        = "setup.wizard"
)
