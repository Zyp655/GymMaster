package seed

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
)

func TestDemoGen_InvariantsAndShowcaseMember(t *testing.T) {
	loc := demoLoc()
	now := time.Date(2026, time.October, 14, 14, 0, 0, 0, loc)
	today := dayOf(now, loc)
	histStart := addDays(today, -150)

	gen := &demoGen{
		r:         demoRand{rand.New(rand.NewSource(20261014))}, // #nosec G404
		loc:       loc,
		now:       now,
		today:     today,
		histStart: histStart,
		ownerID:   1,
		passHash:  "dummy-hash",
		roleIDs: map[string]uint{
			consts.RoleOwner:        1,
			consts.RoleManager:      2,
			consts.RoleReceptionist: 3,
			consts.RoleTrainer:      4,
			consts.RoleMember:       5,
		},
		phones:        make(map[string]bool),
		emails:        make(map[string]bool),
		existingRooms: make(map[string]bool),
		stockBal:      make(map[[2]uint]int),
	}

	gen.branches = []*dBranch{
		{id: 1, idx: 0, code: "HN01", name: "GymMaster Cầu Giấy", open: hhmmToMin("05:30"), close: hhmmToMin("22:00")},
		{id: 2, idx: 1, code: "HN02", name: "GymMaster Hai Bà Trưng", open: hhmmToMin("05:30"), close: hhmmToMin("22:00")},
		{id: 3, idx: 2, code: "HCM01", name: "GymMaster Quận 1", open: hhmmToMin("06:00"), close: hhmmToMin("22:30")},
	}
	for i, def := range demoPlanDefs {
		gen.plans = append(gen.plans, &dPlan{id: uint(i + 1), planDef: def})
	}
	for i, def := range demoPTPkgDefs {
		gen.ptPkgs = append(gen.ptPkgs, &dPTPkg{id: uint(i + 1), ptPkgDef: def})
	}
	for i, def := range demoPromoDefs(today.Year()) {
		st, _ := time.ParseInLocation("2006-01-02", def.Start, loc)
		en, _ := time.ParseInLocation("2006-01-02", def.End, loc)
		en = en.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		gen.promos = append(gen.promos, &dPromo{id: uint(i + 1), def: def, start: st, end: en})
	}
	for i, def := range demoClassTypeDefs {
		gen.classTypes = append(gen.classTypes, &dClassType{id: uint(i + 1), classTypeDef: def})
	}
	gen.catIDs = []uint{1, 2, 3}
	gen.supIDs = []uint{1, 2}
	for i, def := range demoProductDefs {
		gen.products = append(gen.products, &dProduct{
			id:       uint(i + 1),
			def:      def,
			cost:     def.Cost,
			catID:    gen.catIDs[def.Category],
			supplier: gen.supIDs[def.Supplier],
		})
	}

	gen.genStaff()
	gen.genRooms()
	gen.genShifts()
	gen.genMembers()
	gen.addShowcaseMember()
	gen.sortMembers()
	gen.assignMemberUsers()

	gen.genSubscriptions()
	gen.genClasses()
	gen.genPT()
	gen.genCheckins()
	gen.genMetrics()
	gen.genPOS()
	gen.genNotificationsAndAudits()

	require.Len(t, gen.members, demoMemberCount+1)
	require.NotEmpty(t, gen.staff)
	require.NotEmpty(t, gen.subs)
	require.NotEmpty(t, gen.invoices)
	require.NotEmpty(t, gen.schedules)
	require.NotEmpty(t, gen.bookings)
	require.NotEmpty(t, gen.contracts)
	require.NotEmpty(t, gen.ptSessions)
	require.NotEmpty(t, gen.checkins)
	require.NotEmpty(t, gen.metrics)
	require.NotEmpty(t, gen.stockTxs)

	// 1. Kiểm tra tài khoản hội viên mẫu hoivien@gymmaster.vn
	var showcase *dMember
	for _, m := range gen.members {
		if m.email == "hoivien@gymmaster.vn" {
			showcase = m
			break
		}
	}
	require.NotNil(t, showcase, "phải có tài khoản hội viên mẫu hoivien@gymmaster.vn")
	require.NotNil(t, showcase.user, "hội viên mẫu phải được cấp tài khoản cổng hội viên")
	require.GreaterOrEqual(t, len(showcase.subs), 2)
	assert.Equal(t, entity.SubStatusActive, showcase.subs[len(showcase.subs)-1].status)

	// 2. Kiểm tra bất biến tài chính của mọi hóa đơn
	for _, inv := range gen.invoices {
		assert.Equal(t, inv.subtotal-inv.discount, inv.total, "total = subtotal - discount")
		assert.LessOrEqual(t, inv.paid, inv.total, "paid <= total")
		assert.LessOrEqual(t, inv.refunded, inv.paid, "refunded <= paid")
	}

	// 3. Kiểm tra không trùng lịch lớp trong cùng phòng hoặc cùng HLV (EXCLUDE constraint)
	roomIntervals := make(map[*dRoom][]interval)
	trainerIntervals := make(map[*dUser][]interval)
	for _, sch := range gen.schedules {
		if sch.status == entity.ClassScheduleCancelled {
			continue
		}
		assert.False(t, overlaps(roomIntervals[sch.room], sch.start, sch.end), "trùng lịch phòng tập")
		assert.False(t, overlaps(trainerIntervals[sch.trainer], sch.start, sch.end), "trùng lịch HLV dạy lớp")
		roomIntervals[sch.room] = append(roomIntervals[sch.room], interval{sch.start, sch.end})
		trainerIntervals[sch.trainer] = append(trainerIntervals[sch.trainer], interval{sch.start, sch.end})
	}

	// 4. Kiểm tra không trùng buổi PT của cùng HLV (EXCLUDE constraint)
	ptTrainerIntervals := make(map[*dUser][]interval)
	for _, s := range gen.ptSessions {
		if s.status != entity.PTSessionScheduled && s.status != entity.PTSessionCompleted {
			continue
		}
		assert.False(t, overlaps(ptTrainerIntervals[s.c.trainer], s.start, s.end), "trùng lịch PT của HLV")
		ptTrainerIntervals[s.c.trainer] = append(ptTrainerIntervals[s.c.trainer], interval{s.start, s.end})
	}

	// 5. Kiểm tra tồn kho không âm và 4 cặp cảnh báo tồn thấp
	for _, bal := range gen.stockBal {
		assert.GreaterOrEqual(t, bal, 0)
	}
	for _, pair := range demoLowStockPairs {
		b := gen.branches[pair.Branch]
		for _, p := range gen.products {
			if p.def.SKU == pair.SKU {
				qty := gen.stockBal[[2]uint{p.id, b.id}]
				assert.Less(t, qty, p.def.MinStock, "cặp %s tại %s phải dưới min_stock", pair.SKU, b.code)
			}
		}
	}

	// 6. Kiểm tra không trùng ca trực của cùng nhân viên (ex_shifts_staff_overlap)
	staffShiftIntervals := make(map[*dUser][]interval)
	for _, sh := range gen.shifts {
		assert.True(t, sh.end.After(sh.start), "ca trực phải có end > start")
		assert.False(t, overlaps(staffShiftIntervals[sh.staff], sh.start, sh.end), "trùng ca trực nhân sự")
		staffShiftIntervals[sh.staff] = append(staffShiftIntervals[sh.staff], interval{sh.start, sh.end})
	}

	// 7. Kiểm tra bất biến đặt lớp (booked_count <= capacity, không trùng hội viên trên cùng buổi)
	for _, sch := range gen.schedules {
		assert.LessOrEqual(t, sch.booked, sch.capacity, "booked_count <= capacity")
		seenMembers := make(map[*dMember]bool)
		activeBooked := 0
		for _, bk := range sch.bookings {
			if bk.status != entity.BookingStatusCancelled {
				assert.False(t, seenMembers[bk.m], "hội viên không được đặt 2 chỗ active cùng buổi")
				seenMembers[bk.m] = true
			}
			if bk.status == entity.BookingStatusBooked || bk.status == entity.BookingStatusAttended || bk.status == entity.BookingStatusNoShow {
				activeBooked++
			}
		}
		if sch.status != entity.ClassScheduleCancelled {
			assert.Equal(t, sch.booked, activeBooked, "booked_count khớp số booking chiếm chỗ")
		}
	}

	// 8. Kiểm tra chuyển nhượng gói tập (TRANSFERRED + hóa đơn FEE), hóa đơn VOID & PARTIAL, bảo lưu và hoa hồng PT
	transferredSubs := 0
	voidInvoices := 0
	partialInvoices := 0
	feeItems := 0
	for _, s := range gen.subs {
		if s.status == entity.SubStatusTransferred {
			transferredSubs++
			require.NotNil(t, s.transferredTo, "gói TRANSFERRED phải trỏ tới gói mới")
			assert.Equal(t, s, s.transferredTo.prev)
		}
	}
	for _, inv := range gen.invoices {
		if inv.status == entity.InvoiceStatusVoid {
			voidInvoices++
			assert.NotNil(t, inv.voidedAt)
			assert.NotEmpty(t, inv.voidReason)
		}
		if inv.status == entity.InvoiceStatusPartial {
			partialInvoices++
			assert.Greater(t, inv.paid, 0.0)
			assert.Less(t, inv.paid, inv.total)
		}
		for _, it := range inv.items {
			if it.typ == entity.InvoiceItemFee {
				feeItems++
			}
		}
	}
	assert.GreaterOrEqual(t, transferredSubs, 1, "phải có ít nhất 1 ca chuyển nhượng gói tập")
	assert.Equal(t, transferredSubs, feeItems, "mỗi ca chuyển nhượng phải có 1 dòng hóa đơn FEE")
	assert.GreaterOrEqual(t, voidInvoices, 1, "phải có hóa đơn VOID")
	assert.GreaterOrEqual(t, partialInvoices, 1, "phải có hóa đơn PARTIAL")

	freezeStatuses := make(map[string]int)
	for _, fz := range gen.freezes {
		freezeStatuses[fz.status]++
		assert.False(t, fz.end.Before(fz.start), "end_date >= start_date")
	}
	assert.Greater(t, freezeStatuses[entity.FreezeStatusEnded]+freezeStatuses[entity.FreezeStatusApproved], 0)
	assert.Greater(t, freezeStatuses[entity.FreezeStatusPending], 0)
	assert.Greater(t, freezeStatuses[entity.FreezeStatusRejected], 0)

	commStatuses := make(map[string]int)
	for _, cm := range gen.commissions {
		commStatuses[cm.status]++
		assert.Equal(t, roundVND(cm.base*cm.rate/100), cm.amount)
	}
	assert.Greater(t, commStatuses[entity.CommissionPaid], 0)
	assert.Greater(t, commStatuses[entity.CommissionPending], 0)
}

func TestSeedHelpers_BulkAndEventData(t *testing.T) {
	assert.Equal(t, `{}`, string(resolveEventData(nil)))
	inv := &dInvoice{no: "HD26101400001"}
	raw := resolveEventData(map[string]any{"invoice": refInvNo{inv: inv}, "days": 30})
	assert.Contains(t, string(raw), `"HD26101400001"`)
	assert.Contains(t, string(raw), `"days":30`)

	b := newBulk("rooms", "id", "name")
	b.add(1, "Yoga")
	require.Len(t, b.rows, 1)
	assert.Panics(t, func() {
		b.add(2) // thiếu cột
	})
}
