package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
)

func (g *demoGen) classTypeByCode(code string) *dClassType {
	for _, ct := range g.classTypes {
		if ct.Code == code {
			return ct
		}
	}
	return g.classTypes[0]
}

func (g *demoGen) activeSubOn(m *dMember, d time.Time, needClasses bool) *dSub {
	for _, s := range m.subs {
		if d.Before(s.start) || d.After(s.end) {
			continue
		}
		if s.cutoff != nil && !d.Before(*s.cutoff) {
			continue
		}
		if needClasses && !s.plan.Classes {
			continue
		}
		for _, fz := range s.freezes {
			if fz.applied && !d.Before(fz.start) && !d.After(fz.end) {
				return nil
			}
		}
		return s
	}
	return nil
}

func (g *demoGen) genClasses() {
	startWindow := addDays(g.today, -45)
	endWindow := addDays(g.today, 7)

	for bi, b := range g.branches {
		for _, slot := range demoTimetable[bi] {
			ct := g.classTypeByCode(slot.Type)
			rm := b.rooms[slot.Room]
			tr := b.trainers[slot.Trainer]
			seriesID := uuid.NewString()
			startMin := hhmmToMin(slot.Time)

			wset := make(map[int]bool, len(slot.Weekdays))
			for _, w := range slot.Weekdays {
				wset[w] = true
			}

			for d := startWindow; !d.After(endWindow); d = addDays(d, 1) {
				if !wset[isoWeekday(d)] {
					continue
				}
				st := atMin(d, startMin)
				en := st.Add(time.Duration(ct.Duration) * time.Minute)
				if overlaps(rm.busy, st, en) || overlaps(tr.busy, st, en) {
					continue
				}

				capVal := ct.Capacity
				if rm.capacity < capVal {
					capVal = rm.capacity
				}

				status := entity.ClassScheduleScheduled
				cancelReason := ""
				if en.Before(g.now) {
					if g.r.chance(0.03) {
						status = entity.ClassScheduleCancelled
						cancelReason = g.r.pickS(classCancelReasons)
					} else {
						status = entity.ClassScheduleCompleted
					}
				}

				if status != entity.ClassScheduleCancelled {
					rm.busy = append(rm.busy, interval{st, en})
					tr.busy = append(tr.busy, interval{st, en})
				}

				sch := &dSchedule{
					ct:           ct,
					branch:       b,
					room:         rm,
					trainer:      tr,
					start:        st,
					end:          en,
					capacity:     capVal,
					status:       status,
					series:       seriesID,
					cancelReason: cancelReason,
					createdAt:    addDays(startWindow, -7),
					createdBy:    b.manager,
				}
				g.schedules = append(g.schedules, sch)
			}
		}
	}

	// Tạo đặt chỗ cho các buổi lớp
	for _, sch := range g.schedules {
		if sch.status == entity.ClassScheduleCancelled {
			continue
		}
		d := dayOf(sch.start, g.loc)
		var eligible []*dMember
		var showcase *dMember
		for _, m := range g.members {
			if !m.classLover && !m.demo {
				continue
			}
			sub := g.activeSubOn(m, d, true)
			if sub == nil {
				continue
			}
			if sub.plan.Scope == entity.AccessSingleBranch && sub.branch != sch.branch {
				continue
			}
			if overlaps(m.busy, sch.start, sch.end) {
				continue
			}
			if m.demo && sch.branch == m.branch {
				showcase = m
			} else {
				eligible = append(eligible, m)
			}
		}

		target := g.r.between(sch.capacity*45/100, sch.capacity+2)
		if target > len(eligible) {
			target = len(eligible)
		}

		chosen := make([]*dMember, 0, target+1)
		if showcase != nil && g.r.chance(0.35) {
			chosen = append(chosen, showcase)
		}
		usedIdx := make(map[int]bool)
		for len(chosen) < target && len(usedIdx) < len(eligible) {
			idx := g.r.Intn(len(eligible))
			if usedIdx[idx] {
				continue
			}
			usedIdx[idx] = true
			chosen = append(chosen, eligible[idx])
		}

		for _, m := range chosen {
			sub := g.activeSubOn(m, d, true)
			bookedAt := sch.start.Add(-time.Duration(g.r.between(180, 60*72)) * time.Minute)
			if bookedAt.After(g.now) {
				bookedAt = g.now.Add(-time.Duration(g.r.between(15, 180)) * time.Minute)
			}
			src := entity.BookingSourceCounter
			var creator *dUser = g.recep(sch.branch)
			if m.user != nil || g.r.chance(0.55) {
				src = entity.BookingSourcePortal
				creator = m.user
			}

			bk := &dBooking{
				sch:       sch,
				m:         m,
				sub:       sub,
				bookedAt:  bookedAt,
				source:    src,
				createdBy: creator,
				updatedAt: bookedAt,
			}

			if sch.status == entity.ClassScheduleCompleted {
				r := g.r.Float64()
				switch {
				case r < 0.08:
					cAt := sch.start.Add(-time.Duration(g.r.between(150, 600)) * time.Minute)
					bk.status = entity.BookingStatusCancelled
					bk.cancelledAt = &cAt
					bk.updatedAt = cAt
				case sch.booked < sch.capacity && r < 0.88:
					aAt := sch.start.Add(-time.Duration(g.r.between(5, 25)) * time.Minute)
					bk.status = entity.BookingStatusAttended
					bk.attendedAt = &aAt
					bk.updatedAt = aAt
					sch.booked++
					m.busy = append(m.busy, interval{sch.start, sch.end})
				case sch.booked < sch.capacity:
					bk.status = entity.BookingStatusNoShow
					bk.updatedAt = sch.end.Add(5 * time.Minute)
					sch.booked++
					if !m.demo {
						m.noShows = append(m.noShows, sch.end)
					}
				default:
					sch.waitPos++
					bk.status = entity.BookingStatusWaitlist
					bk.waitPos = sch.waitPos
				}
			} else {
				if sch.booked < sch.capacity {
					bk.status = entity.BookingStatusBooked
					sch.booked++
					m.busy = append(m.busy, interval{sch.start, sch.end})
				} else {
					sch.waitPos++
					bk.status = entity.BookingStatusWaitlist
					bk.waitPos = sch.waitPos
				}
			}

			sch.bookings = append(sch.bookings, bk)
			g.bookings = append(g.bookings, bk)
		}
	}
}

func (g *demoGen) genPT() {
	pkgWeights := []float64{50, 35, 15}
	for idx, m := range g.members {
		if !m.demo && idx%6 != 0 {
			continue
		}
		if len(m.subs) == 0 {
			continue
		}
		b := m.branch
		tr := b.trainers[idx%len(b.trainers)]
		pkg := g.ptPkgs[g.r.weighted(pkgWeights)]
		if m.demo {
			pkg = g.ptPkgs[0] // PT10
			tr = b.trainers[0]
		}

		lastSub := m.subs[len(m.subs)-1]
		startDay := maxTime(lastSub.start, addDays(g.today, -40))
		if m.demo {
			startDay = addDays(g.today, -28)
		}
		if startDay.After(g.today) {
			startDay = g.today
		}
		endDay := addDays(startDay, pkg.Validity-1)
		issuedAt := atMin(startDay, g.r.between(9*60, 18*60))
		if issuedAt.After(g.now) {
			issuedAt = g.now.Add(-2 * time.Hour)
		}
		seller := g.pickSeller(b)

		inv := &dInvoice{
			member:    m,
			branch:    b,
			subtotal:  pkg.Price,
			total:     pkg.Price,
			paid:      pkg.Price,
			status:    entity.InvoiceStatusPaid,
			source:    entity.InvoiceSourceCounter,
			note:      fmt.Sprintf("Hợp đồng %s – HLV %s", pkg.Name, tr.name),
			issuedBy:  seller,
			issuedAt:  issuedAt,
			updatedAt: issuedAt,
		}

		c := &dContract{
			m:           m,
			trainer:     tr,
			pkg:         pkg,
			branch:      b,
			inv:         inv,
			total:       pkg.Sessions,
			minutes:     pkg.Minutes,
			start:       startDay,
			end:         endDay,
			price:       pkg.Price,
			status:      entity.PTContractActive,
			soldBy:      seller,
			note:        fmt.Sprintf("HLV phụ trách: %s", tr.name),
			createdAt:   issuedAt,
			updatedAt:   issuedAt,
			activatedAt: issuedAt,
		}

		inv.items = append(inv.items, &dItem{
			typ:         entity.InvoiceItemPT,
			refContract: c,
			desc:        fmt.Sprintf("%s (%d buổi x %d phút)", pkg.Name, pkg.Sessions, pkg.Minutes),
			qty:         1,
			unit:        pkg.Price,
			amount:      pkg.Price,
		})
		pm := &dPayment{
			inv:       inv,
			amount:    pkg.Price,
			method:    entity.PaymentTransfer,
			reference: fmt.Sprintf("PT%s%04d", issuedAt.Format("060102"), g.r.between(1000, 9999)),
			at:        issuedAt.Add(time.Minute),
			by:        seller,
			note:      "Thanh toán gói PT",
		}
		inv.payments = append(inv.payments, pm)

		g.invoices = append(g.invoices, inv)
		g.payments = append(g.payments, pm)
		g.contracts = append(g.contracts, c)

		// Xếp các buổi PT cho hợp đồng này
		firstSessDay := maxTime(c.start, dayOf(c.activatedAt, g.loc))
		lastSessDay := minTime(c.end, addDays(g.today, 6))
		hours := []int{7 * 60, 8*60 + 30, 10 * 60, 15 * 60, 16*60 + 30, 19 * 60}

		for d := firstSessDay; !d.After(lastSessDay) && c.used < c.total; d = addDays(d, g.r.between(2, 4)) {
			if m.demo && len(c.sessions) >= 6 {
				break
			}
			slotMin := hours[g.r.Intn(len(hours))]
			st := atMin(d, slotMin)
			en := st.Add(time.Duration(c.minutes) * time.Minute)
			if overlaps(tr.busy, st, en) || overlaps(m.busy, st, en) {
				continue
			}

			sess := &dPTSession{
				c:         c,
				start:     st,
				end:       en,
				source:    entity.PTSourcePortal,
				createdBy: tr,
				createdAt: st.Add(-24 * time.Hour),
			}

			if en.Before(g.now) {
				if !m.demo && g.r.chance(0.06) {
					cAt := st.Add(-14 * time.Hour)
					sess.status = entity.PTSessionCancelled
					sess.cancelledAt = &cAt
					sess.cancelReason = "Hội viên báo bận công việc đột xuất"
				} else {
					compAt := en.Add(5 * time.Minute)
					sess.status = entity.PTSessionCompleted
					sess.workout = g.r.pickS(ptWorkoutNotes)
					sess.tnote = g.r.pickS(ptTrainerNotes)
					sess.completedAt = &compAt
					c.used++
					c.updatedAt = compAt
					tr.busy = append(tr.busy, interval{st, en})
					m.busy = append(m.busy, interval{st, en})

					base := roundVND(c.price / float64(c.total))
					amt := roundVND(base * tr.rate / 100)
					period := st.Format("2006-01")
					commStatus := entity.CommissionPending
					var paidAt *time.Time
					var paidBy *uint
					if period < g.today.Format("2006-01") {
						commStatus = entity.CommissionPaid
						pTime := atMin(time.Date(g.today.Year(), g.today.Month(), 5, 0, 0, 0, 0, g.loc), 10*60)
						if pTime.After(g.now) {
							pTime = g.now.Add(-24 * time.Hour)
						}
						paidAt = &pTime
						paidBy = ptrUint(g.ownerID)
					}
					g.commissions = append(g.commissions, &dCommission{
						s:         sess,
						base:      base,
						rate:      tr.rate,
						amount:    amt,
						period:    period,
						status:    commStatus,
						paidAt:    paidAt,
						paidBy:    paidBy,
						createdAt: compAt,
					})
				}
			} else {
				sess.status = entity.PTSessionScheduled
				tr.busy = append(tr.busy, interval{st, en})
				m.busy = append(m.busy, interval{st, en})
			}

			c.sessions = append(c.sessions, sess)
			g.ptSessions = append(g.ptSessions, sess)
		}

		// Đảm bảo hội viên mẫu có ít nhất 1 buổi PT sắp tới
		if m.demo {
			hasUpcoming := false
			for _, s := range c.sessions {
				if s.status == entity.PTSessionScheduled {
					hasUpcoming = true
					break
				}
			}
			if !hasUpcoming && c.used < c.total {
				tom := addDays(g.today, 1)
				st := atMin(tom, 18*60)
				en := st.Add(time.Duration(c.minutes) * time.Minute)
				if !overlaps(tr.busy, st, en) && !overlaps(m.busy, st, en) {
					tr.busy = append(tr.busy, interval{st, en})
					m.busy = append(m.busy, interval{st, en})
					sess := &dPTSession{
						c:         c,
						start:     st,
						end:       en,
						status:    entity.PTSessionScheduled,
						source:    entity.PTSourcePortal,
						createdBy: m.user,
						createdAt: g.now.Add(-3 * time.Hour),
					}
					c.sessions = append(c.sessions, sess)
					g.ptSessions = append(g.ptSessions, sess)
				}
			}
		}

		if c.used >= c.total {
			c.status = entity.PTContractCompleted
		} else if c.end.Before(g.today) {
			c.status = entity.PTContractExpired
		}
	}
}

func (g *demoGen) pickCheckinMinute(pref int) int {
	switch pref {
	case 0:
		return g.r.between(6*60, 9*60+30)
	case 1:
		return g.r.between(11*60+30, 14*60)
	default:
		return g.r.between(17*60, 20*60+30)
	}
}

func (g *demoGen) genCheckins() {
	checkStart := addDays(g.today, -60)
	for _, m := range g.members {
		if len(m.subs) == 0 {
			continue
		}
		dailyProb := m.freq / 7.0
		for d := checkStart; !d.After(g.today); d = addDays(d, 1) {
			if d.Before(dayOf(m.createdAt, g.loc)) {
				continue
			}
			if !g.r.chance(dailyProb) {
				continue
			}
			inMin := g.pickCheckinMinute(m.pref)
			inAt := atMin(d, inMin)
			if inAt.After(g.now) {
				if !d.Equal(g.today) {
					continue
				}
				inAt = g.now.Add(-time.Duration(g.r.between(10, 90)) * time.Minute)
			}

			sub := g.activeSubOn(m, d, false)
			if sub == nil {
				// Ghi nhận một vài lượt quét bị từ chối (gói hết hạn hoặc đang bảo lưu)
				if g.r.chance(0.08) {
					deny := entity.CheckinDenyNoActiveSubscription
					last := m.subs[len(m.subs)-1]
					if last.status == entity.SubStatusFrozen {
						deny = entity.CheckinDenySubscriptionFrozen
					}
					g.checkins = append(g.checkins, &dCheckin{
						m:      m,
						branch: m.branch,
						in:     inAt,
						method: entity.CheckinMethodQR,
						result: entity.CheckinDenied,
						deny:   deny,
						staff:  g.recep(m.branch),
					})
				}
				continue
			}

			if sub.plan.Type == entity.PlanTypeSession {
				if sub.remaining <= 0 {
					if !sub.deniedEmpty {
						sub.deniedEmpty = true
						g.checkins = append(g.checkins, &dCheckin{
							m:      m,
							branch: m.branch,
							sub:    sub,
							in:     inAt,
							method: entity.CheckinMethodQR,
							result: entity.CheckinDenied,
							deny:   entity.CheckinDenyNoSessionsLeft,
							staff:  g.recep(m.branch),
						})
					}
					continue
				}
				sub.remaining--
			}

			durMin := g.r.between(50, 110)
			outAt := inAt.Add(time.Duration(durMin) * time.Minute)
			var outPtr *time.Time
			// Để các lượt vừa vào trong 75 phút gần đây chưa checkout ("Đang trong phòng tập")
			if outAt.Before(g.now) && (!d.Equal(g.today) || g.now.Sub(inAt) > 75*time.Minute) {
				outPtr = &outAt
			}

			method := entity.CheckinMethodQR
			if g.r.chance(0.12) {
				method = entity.CheckinMethodManual
			}
			g.checkins = append(g.checkins, &dCheckin{
				m:      m,
				branch: m.branch,
				sub:    sub,
				in:     inAt,
				out:    outPtr,
				method: method,
				result: entity.CheckinAllowed,
				staff:  g.recep(m.branch),
			})
		}
	}
}

func (g *demoGen) genMetrics() {
	for idx, m := range g.members {
		if !m.demo && idx%3 != 0 {
			continue
		}
		height := roundTo(g.r.normal(170, 5.5), 0.5)
		weight := roundTo(g.r.normal(70, 9), 0.1)
		fat := roundTo(g.r.normal(22, 4), 0.1)
		muscle := roundTo(weight*(1-fat/100)*0.48, 0.1)
		waist := roundTo(g.r.normal(82, 6), 0.5)
		chest := roundTo(g.r.normal(94, 5), 0.5)
		hip := roundTo(g.r.normal(95, 5), 0.5)

		if m.gender == "FEMALE" {
			height = roundTo(g.r.normal(159, 4.5), 0.5)
			weight = roundTo(g.r.normal(54, 6), 0.1)
			fat = roundTo(g.r.normal(26, 4), 0.1)
			muscle = roundTo(weight*(1-fat/100)*0.42, 0.1)
			waist = roundTo(g.r.normal(68, 5), 0.5)
			chest = roundTo(g.r.normal(84, 4), 0.5)
			hip = roundTo(g.r.normal(91, 4), 0.5)
		}
		if m.demo {
			height, weight, fat, muscle, waist, chest, hip = 173, 74.5, 21.8, 32.4, 83.0, 96.0, 96.5
		}

		tr := m.branch.trainers[idx%len(m.branch.trainers)]
		count := g.r.between(2, 4)
		if m.demo {
			count = 4
		}
		spanDays := max(10, daysBetween(dayOf(m.createdAt, g.loc), g.today))
		step := max(7, spanDays/count)

		for k := 0; k < count; k++ {
			mDay := addDays(dayOf(m.createdAt, g.loc), k*step)
			if mDay.After(g.today) {
				mDay = g.today
			}
			at := atMin(mDay, g.r.between(8*60, 19*60))
			if at.After(g.now) {
				at = g.now.Add(-time.Hour)
			}
			note := "Đo chỉ số định kỳ InBody"
			if k == 0 {
				note = "Đo chỉ số đầu vào khi đăng ký"
			}
			g.metrics = append(g.metrics, &dMetric{
				m:      m,
				at:     at,
				weight: math.Max(42, roundTo(weight-float64(k)*0.8, 0.1)),
				height: height,
				fat:    math.Max(10, roundTo(fat-float64(k)*0.6, 0.1)),
				muscle: roundTo(muscle+float64(k)*0.3, 0.1),
				waist:  math.Max(58, roundTo(waist-float64(k)*0.7, 0.5)),
				chest:  roundTo(chest+float64(k)*0.3, 0.5),
				hip:    roundTo(hip-float64(k)*0.2, 0.5),
				note:   note,
				by:     tr,
			})
		}
	}
}

func (g *demoGen) addStockTx(p *dProduct, b *dBranch, typ string, delta int, unitCost float64, inv *dInvoice, sup *uint, voucher, note string, by *dUser, at time.Time) {
	key := [2]uint{p.id, b.id}
	bal := g.stockBal[key] + delta
	if bal < 0 {
		bal = 0
	}
	g.stockBal[key] = bal
	g.stockTxs = append(g.stockTxs, &dStockTx{
		p:          p,
		branch:     b,
		typ:        typ,
		qty:        delta,
		balance:    bal,
		unitCost:   unitCost,
		inv:        inv,
		supplierID: sup,
		voucher:    voucher,
		note:       note,
		by:         by,
		at:         at,
	})
}

func (g *demoGen) genPOS() {
	// 1. Nhập kho đầu kỳ tại cả 3 chi nhánh
	initAt := atMin(g.histStart, 8*60+30)
	for bi, b := range g.branches {
		voucher := fmt.Sprintf("NK-%s-001", b.code)
		for _, p := range g.products {
			qty := p.def.Target[bi] * 2
			sup := ptrUint(p.supplier)
			g.addStockTx(p, b, entity.StockImport, qty, p.cost, nil, sup, voucher, "Nhập kho đầu kỳ", b.manager, initAt)
		}
	}

	// 2. Bán hàng tại quầy (POS) xuyên suốt 60 ngày gần đây + bổ sung tồn kho khi cạn
	prodWeights := make([]float64, len(g.products))
	for i, p := range g.products {
		prodWeights[i] = p.def.Weight
	}

	posStart := addDays(g.today, -60)
	restockSeq := 2
	for d := posStart; !d.After(g.today); d = addDays(d, 1) {
		for bi, b := range g.branches {
			ordersToday := g.r.between(2, 6)
			for oi := 0; oi < ordersToday; oi++ {
				at := atMin(d, g.r.between(b.open+30, b.close-30))
				if at.After(g.now) {
					if !d.Equal(g.today) {
						continue
					}
					at = g.now.Add(-time.Duration(g.r.between(5, 120)) * time.Minute)
				}
				seller := g.recep(b)
				var buyer *dMember
				if g.r.chance(0.55) {
					buyer = g.members[g.r.Intn(len(g.members))]
				}

				itemCount := g.r.between(1, 3)
				usedProd := make(map[uint]bool)
				inv := &dInvoice{
					member:    buyer,
					branch:    b,
					status:    entity.InvoiceStatusPaid,
					source:    entity.InvoiceSourcePOS,
					note:      "Bán lẻ tại quầy POS",
					issuedBy:  seller,
					issuedAt:  at,
					updatedAt: at,
				}

				for ic := 0; ic < itemCount; ic++ {
					p := g.products[g.r.weighted(prodWeights)]
					if usedProd[p.id] {
						continue
					}
					usedProd[p.id] = true
					qty := g.r.between(1, 2)
					key := [2]uint{p.id, b.id}
					if g.stockBal[key] < qty+p.def.MinStock {
						addQty := max(p.def.Target[bi], p.def.Case*2)
						vNo := fmt.Sprintf("NK-%s-%03d", b.code, restockSeq)
						restockSeq++
						g.addStockTx(p, b, entity.StockImport, addQty, p.cost, nil, ptrUint(p.supplier), vNo, "Nhập bổ sung hàng POS", b.manager, at.Add(-time.Hour))
					}

					unitPrice := p.def.Sell
					if buyer != nil && p.def.Member > 0 {
						unitPrice = p.def.Member
					}
					amt := roundVND(unitPrice * float64(qty))
					inv.subtotal += amt
					inv.items = append(inv.items, &dItem{
						typ:        entity.InvoiceItemProduct,
						refProduct: p,
						desc:       p.def.Name,
						qty:        qty,
						unit:       unitPrice,
						amount:     amt,
					})
					g.addStockTx(p, b, entity.StockSale, -qty, p.cost, inv, nil, "", "Xuất bán POS", seller, at)
				}

				if len(inv.items) == 0 {
					continue
				}
				inv.total = inv.subtotal
				inv.paid = inv.total
				method := g.r.pickS([]string{entity.PaymentCash, entity.PaymentTransfer, entity.PaymentCard})
				pm := &dPayment{
					inv:    inv,
					amount: inv.total,
					method: method,
					at:     at,
					by:     seller,
					note:   "Thanh toán POS",
				}
				inv.payments = append(inv.payments, pm)
				g.invoices = append(g.invoices, inv)
				g.payments = append(g.payments, pm)
			}
		}
	}

	// 3. Một phiếu điều chuyển kho (TRANSFER_OUT / TRANSFER_IN) giữa HN01 và HN02
	tfAt := atMin(addDays(g.today, -7), 10*60)
	bFrom, bTo := g.branches[0], g.branches[1]
	pTf := g.products[0] // AQUA-500
	if g.stockBal[[2]uint{pTf.id, bFrom.id}] > 30 {
		g.addStockTx(pTf, bFrom, entity.StockTransferOut, -24, pTf.cost, nil, nil, "DC-HN01-001", "Điều chuyển sang Hai Bà Trưng", bFrom.manager, tfAt)
		g.addStockTx(pTf, bTo, entity.StockTransferIn, 24, pTf.cost, nil, nil, "DC-HN01-001", "Nhận điều chuyển từ Cầu Giấy", bTo.manager, tfAt.Add(time.Minute))
	}

	// 4. Đưa 4 cặp (SKU, chi nhánh) trong demoLowStockPairs về mức cảnh báo tồn thấp
	adjAt := g.now.Add(-45 * time.Minute)
	for i, pair := range demoLowStockPairs {
		b := g.branches[pair.Branch]
		for _, p := range g.products {
			if p.def.SKU != pair.SKU {
				continue
			}
			key := [2]uint{p.id, b.id}
			cur := g.stockBal[key]
			targetLow := max(1, p.def.MinStock-2)
			if cur > targetLow {
				diff := targetLow - cur
				vNo := fmt.Sprintf("KK-%s-%02d", b.code, i+1)
				g.addStockTx(p, b, entity.StockAdjust, diff, p.cost, nil, nil, vNo, "Kiểm kê cuối kỳ – tồn thấp cần nhập thêm", b.manager, adjAt)
			}
		}
	}
}

func (g *demoGen) genNotificationsAndAudits() {
	// Thông báo cho OWNER và các quản lý chi nhánh
	g.notifs = append(g.notifs,
		&dNotif{
			userID: g.ownerID,
			typ:    "LOW_STOCK",
			title:  "Cảnh báo 4 mặt hàng dưới định mức tồn kho",
			body:   "Có 4 mặt hàng tại Cầu Giấy, Hai Bà Trưng và Quận 1 đang dưới mức tồn tối thiểu. Vui lòng kiểm tra và nhập bổ sung.",
			link:   "/app/inventory",
			at:     g.now.Add(-40 * time.Minute),
		},
		&dNotif{
			userID: g.ownerID,
			typ:    "FREEZE_REQUEST",
			title:  "Có yêu cầu bảo lưu gói tập đang chờ duyệt",
			body:   "Hội viên vừa gửi yêu cầu bảo lưu gói tập. Vui lòng xem xét và phê duyệt.",
			link:   "/app/subscription-freezes",
			at:     g.now.Add(-2 * time.Hour),
		},
		&dNotif{
			userID: g.ownerID,
			typ:    "SYSTEM",
			title:  "Báo cáo tổng hợp ngày đã sẵn sàng",
			body:   "Hệ thống đã hoàn tất tổng hợp số liệu doanh thu và lượt check-in hôm qua.",
			link:   "/app/reports",
			readAt: ptrTime(g.now.Add(-3 * time.Hour)),
			at:     g.now.Add(-6 * time.Hour),
		},
	)

	// Thông báo cho tài khoản hội viên mẫu (hoivien@gymmaster.vn)
	for _, m := range g.members {
		if !m.demo || m.user == nil {
			continue
		}
		g.notifs = append(g.notifs,
			&dNotif{
				user:  m.user,
				typ:   "PT_SCHEDULED",
				title: "Lịch tập PT ngày mai lúc 18:00",
				body:  "Bạn có buổi tập PT 1 kèm 1 cùng HLV Lê Hoàng Nam tại GymMaster Cầu Giấy vào 18:00 ngày mai.",
				link:  "/member/pt-sessions",
				at:    g.now.Add(-2 * time.Hour),
			},
			&dNotif{
				user:   m.user,
				typ:    "FREEZE_RESULT",
				title:  "Yêu cầu bảo lưu gói tập đã được duyệt",
				body:   "Yêu cầu bảo lưu Gói 3 tháng của bạn đã được Quản lý chi nhánh phê duyệt.",
				link:   "/member/subscriptions",
				readAt: ptrTime(addDays(g.today, -20)),
				at:     addDays(g.today, -25),
			},
		)
	}

	// Nhật ký thao tác (audit_logs) mẫu
	if len(g.subs) > 0 && len(g.invoices) > 0 {
		firstSub := g.subs[len(g.subs)-1]
		firstInv := g.invoices[len(g.invoices)-1]
		g.audits = append(g.audits,
			&dAudit{
				actor:    ptrUint(g.ownerID),
				action:   "SELL",
				entity:   "subscription",
				entityID: func() uint { return firstSub.id },
				before:   nil,
				after: func() map[string]any {
					return map[string]any{
						"member":    firstSub.m.name,
						"plan_code": firstSub.plan.Code,
						"price":     firstSub.price,
						"status":    firstSub.status,
					}
				},
				ip: "127.0.0.1",
				at: firstSub.createdAt,
			},
			&dAudit{
				actor:    ptrUint(g.ownerID),
				action:   "CREATE",
				entity:   "invoice",
				entityID: func() uint { return firstInv.id },
				before:   map[string]any{"status": "DRAFT"},
				after: func() map[string]any {
					return map[string]any{
						"invoice_no": firstInv.no,
						"total":      firstInv.total,
						"status":     firstInv.status,
					}
				},
				ip: "127.0.0.1",
				at: firstInv.issuedAt,
			},
		)
	}
}

func (g *demoGen) persistAll(ctx context.Context, tx *gorm.DB) error {
	// 1. Gán ID cho users (staff + memberUsers)
	var maxUserID uint
	_ = tx.WithContext(ctx).Raw("SELECT COALESCE(MAX(id), 0) FROM users").Scan(&maxUserID).Error
	nextUserID := maxUserID + 1

	usersBulk := newBulk("users", "id", "email", "phone", "password_hash", "full_name", "status", "last_login_at", "created_at", "updated_at")
	userRolesBulk := newBulk("user_roles", "user_id", "role_id", "branch_id", "created_at")
	staffBulk := newBulk("staff_profiles", "user_id", "staff_code", "position", "gender", "dob", "id_card", "address", "hire_date", "base_salary", "commission_rate", "note", "created_at", "updated_at")
	trainerBulk := newBulk("trainer_profiles", "user_id", "level", "specialties", "bio", "certifications", "years_experience", "created_at", "updated_at")

	for idx, u := range g.staff {
		u.id = nextUserID
		nextUserID++
		u.staffCode = fmt.Sprintf("NV%04d", idx+1)
		usersBulk.add(u.id, u.email, u.phone, g.passHash, u.name, consts.StatusActive, nTime(u.lastLogin), u.createdAt, u.createdAt)
		userRolesBulk.add(u.id, g.roleIDs[u.role], nUint(ptrUint(u.branch.id)), u.createdAt)
		staffBulk.add(
			u.id,
			u.staffCode,
			u.position,
			u.gender,
			pgDate(u.dob),
			g.idCard(u.branch, u.gender, u.dob),
			g.address(u.branch),
			pgDate(u.hire),
			u.salary,
			u.rate,
			"",
			u.createdAt,
			u.createdAt,
		)
		if u.position == "TRAINER" {
			specs, _ := entity.StaffTextArray(u.def.Specialties).Value()
			trainerBulk.add(u.id, u.def.Level, specs, u.def.Bio, u.def.Certs, u.def.Years, u.createdAt, u.createdAt)
		}
	}

	for _, u := range g.memberUsers {
		u.id = nextUserID
		nextUserID++
		fullName := u.name
		if u.member != nil && u.member.name != "" {
			fullName = u.member.name
		}
		usersBulk.add(u.id, u.email, u.phone, g.passHash, fullName, consts.StatusActive, nTime(u.lastLogin), u.createdAt, u.createdAt)
		userRolesBulk.add(u.id, g.roleIDs[consts.RoleMember], nil, u.createdAt)
	}

	if err := usersBulk.flush(ctx, tx); err != nil {
		return err
	}
	if err := userRolesBulk.flush(ctx, tx); err != nil {
		return err
	}
	if err := staffBulk.flush(ctx, tx); err != nil {
		return err
	}
	if err := trainerBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 2. Rooms & Shifts
	var maxRoomID uint
	_ = tx.WithContext(ctx).Raw("SELECT COALESCE(MAX(id), 0) FROM rooms").Scan(&maxRoomID).Error
	roomsBulk := newBulk("rooms", "id", "branch_id", "name", "capacity", "description", "status", "created_by")
	for i, rm := range g.rooms {
		rm.id = maxRoomID + uint(i+1)
		desc := rm.desc
		if rm.branch.name != "" && desc == "" {
			desc = rm.branch.name
		}
		roomsBulk.add(rm.id, rm.branch.id, rm.name, rm.capacity, desc, consts.StatusActive, g.ownerID)
	}
	if err := roomsBulk.flush(ctx, tx); err != nil {
		return err
	}

	shiftsBulk := newBulk("shifts", "branch_id", "staff_id", "start_at", "end_at", "note", "created_by")
	for _, sh := range g.shifts {
		shiftsBulk.add(sh.branch.id, sh.staff.id, sh.start, sh.end, sh.note, sh.branch.manager.id)
	}
	if err := shiftsBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 3. Members
	membersBulk := newBulk("members",
		"id", "user_id", "member_code", "full_name", "gender", "dob", "phone", "email",
		"id_card", "address", "emergency_contact_name", "emergency_contact_phone",
		"home_branch_id", "source", "status", "booking_blocked_until", "note",
		"created_at", "updated_at", "created_by",
	)
	winStart := g.now.Add(-30 * 24 * time.Hour)
	for i, m := range g.members {
		m.id = uint(i + 1)
		m.code = fmt.Sprintf("GM%06d", m.id)
		var uid *uint
		if m.user != nil {
			uid = ptrUint(m.user.id)
		}
		var blockedUntil *time.Time
		recentNS := 0
		var lastNS time.Time
		for _, ns := range m.noShows {
			if ns.After(winStart) {
				recentNS++
				if ns.After(lastNS) {
					lastNS = ns
				}
			}
		}
		if recentNS >= 3 {
			bu := lastNS.Add(7 * 24 * time.Hour)
			if bu.After(g.now) {
				blockedUntil = &bu
			}
		}
		membersBulk.add(
			m.id, nUint(uid), m.code, m.name, m.gender, pgDate(m.dob), m.phone, m.email,
			m.idCard, m.address, m.ecName, m.ecPhone,
			m.branch.id, m.source, m.status, nTime(blockedUntil), m.note,
			m.createdAt, m.createdAt, m.creator.id,
		)
	}
	if err := membersBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 4. Invoices (sắp xếp theo thời gian để sinh mã HDyyMMdd00001 chuẩn xác)
	sortInvoicesChronologically(g.invoices)
	invBulk := newBulk("invoices",
		"id", "invoice_no", "member_id", "branch_id", "subtotal", "discount", "total",
		"paid_amount", "refunded_amount", "status", "source", "promotion_id", "note",
		"issued_by", "issued_at", "voided_at", "voided_by", "void_reason", "created_at", "updated_at",
	)
	for i, inv := range g.invoices {
		inv.id = uint(i + 1)
		inv.no = fmt.Sprintf("HD%s%05d", inv.issuedAt.In(g.loc).Format("060102"), inv.id)
		var mid, pid, vby *uint
		if inv.member != nil {
			mid = ptrUint(inv.member.id)
		}
		if inv.promo != nil {
			pid = ptrUint(inv.promo.id)
		}
		if inv.voidedAt != nil {
			vby = ptrUint(inv.branch.manager.id)
		}
		invBulk.add(
			inv.id, inv.no, nUint(mid), inv.branch.id, inv.subtotal, inv.discount, inv.total,
			inv.paid, inv.refunded, inv.status, inv.source, nUint(pid), inv.note,
			inv.issuedBy.id, inv.issuedAt, nTime(inv.voidedAt), nUint(vby), inv.voidReason,
			inv.issuedAt, inv.updatedAt,
		)
	}
	if err := invBulk.flush(ctx, tx); err != nil {
		return err
	}

	// Cập nhật used_count cho promotions
	for _, p := range g.promos {
		if p.used > 0 {
			if err := tx.WithContext(ctx).Model(&entity.Promotion{}).Where("id = ?", p.id).UpdateColumn("used_count", p.used).Error; err != nil {
				return err
			}
		}
	}

	// 5. Subscriptions, Freezes, Events
	for i, s := range g.subs {
		s.id = uint(i + 1)
	}
	subsBulk := newBulk("subscriptions",
		"id", "member_id", "plan_id", "branch_id", "invoice_id", "start_date", "end_date",
		"total_sessions", "remaining_sessions", "price", "status", "freeze_days_used",
		"freeze_times_used", "previous_id", "sold_by", "note", "created_at", "updated_at",
	)
	for _, s := range g.subs {
		var totS, remS any
		if s.plan.Type == entity.PlanTypeSession {
			totS = s.total
			remS = s.remaining
		}
		var prevID *uint
		if s.prev != nil {
			prevID = ptrUint(s.prev.id)
		}
		var invID *uint
		if s.inv != nil {
			invID = ptrUint(s.inv.id)
		}
		note := s.note
		if s.transferredTo != nil && s.transferredTo.m != nil && note == s.plan.Name {
			note = fmt.Sprintf("Đã chuyển nhượng sang %s", s.transferredTo.m.name)
		} else if s.next != nil && s.status == entity.SubStatusExpired && s.activatedAt != nil {
			note = s.plan.Name
		}
		subsBulk.add(
			s.id, s.m.id, s.plan.id, s.branch.id, nUint(invID), pgDate(s.start), pgDate(s.end),
			totS, remS, s.price, s.status, s.freezeDays,
			s.freezeTimes, nUint(prevID), s.soldBy.id, note, s.createdAt, s.updatedAt,
		)
	}
	if err := subsBulk.flush(ctx, tx); err != nil {
		return err
	}

	freezesBulk := newBulk("subscription_freezes",
		"id", "subscription_id", "start_date", "end_date", "reason", "status",
		"requested_by", "reviewed_by", "reviewed_at", "review_note", "created_at",
	)
	for i, fz := range g.freezes {
		fz.id = uint(i + 1)
		var reqBy, revBy *uint
		if fz.requestedBy != nil {
			reqBy = ptrUint(fz.requestedBy.id)
		}
		if fz.reviewedBy != nil {
			revBy = ptrUint(fz.reviewedBy.id)
		}
		freezesBulk.add(
			fz.id, fz.sub.id, pgDate(fz.start), pgDate(fz.end), fz.reason, fz.status,
			nUint(reqBy), nUint(revBy), nTime(fz.reviewedAt), fz.reviewNote, fz.createdAt,
		)
	}
	if err := freezesBulk.flush(ctx, tx); err != nil {
		return err
	}

	eventsBulk := newBulk("subscription_events", "id", "subscription_id", "event_type", "data", "actor_id", "created_at")
	for i, ev := range g.events {
		ev.id = uint(i + 1)
		var actID *uint
		if ev.actor != nil {
			actID = ptrUint(ev.actor.id)
		}
		eventsBulk.add(ev.id, ev.sub.id, ev.typ, string(resolveEventData(ev.data)), nUint(actID), ev.at)
	}
	if err := eventsBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 6. PT Contracts, Sessions, Commissions
	contractsBulk := newBulk("pt_contracts",
		"id", "member_id", "trainer_id", "pt_package_id", "branch_id", "invoice_id",
		"total_sessions", "used_sessions", "session_minutes", "start_date", "end_date",
		"price", "status", "sold_by", "note", "created_at", "updated_at",
	)
	for i, c := range g.contracts {
		c.id = uint(i + 1)
		contractsBulk.add(
			c.id, c.m.id, c.trainer.id, c.pkg.id, c.branch.id, c.inv.id,
			c.total, c.used, c.minutes, pgDate(c.start), pgDate(c.end),
			c.price, c.status, c.soldBy.id, c.note, c.createdAt, c.updatedAt,
		)
	}
	if err := contractsBulk.flush(ctx, tx); err != nil {
		return err
	}

	ptSessBulk := newBulk("pt_sessions",
		"id", "contract_id", "trainer_id", "member_id", "branch_id", "start_at", "end_at",
		"status", "workout_note", "trainer_note", "completed_at", "cancelled_at",
		"cancel_reason", "source", "created_by", "created_at", "updated_at",
	)
	for i, s := range g.ptSessions {
		s.id = uint(i + 1)
		var cby *uint
		if s.createdBy != nil {
			cby = ptrUint(s.createdBy.id)
		}
		ptSessBulk.add(
			s.id, s.c.id, s.c.trainer.id, s.c.m.id, s.c.branch.id, s.start, s.end,
			s.status, s.workout, s.tnote, nTime(s.completedAt), nTime(s.cancelledAt),
			s.cancelReason, s.source, nUint(cby), s.createdAt, s.createdAt,
		)
	}
	if err := ptSessBulk.flush(ctx, tx); err != nil {
		return err
	}

	commBulk := newBulk("trainer_commissions",
		"id", "trainer_id", "branch_id", "source_type", "source_id", "base_amount",
		"rate", "amount", "period", "status", "paid_at", "paid_by", "created_at",
	)
	for i, cm := range g.commissions {
		cm.id = uint(i + 1)
		commBulk.add(
			cm.id, cm.s.c.trainer.id, cm.s.c.branch.id, entity.CommissionSourcePTSession, cm.s.id,
			cm.base, cm.rate, cm.amount, cm.period, cm.status, nTime(cm.paidAt), nUint(cm.paidBy), cm.createdAt,
		)
	}
	if err := commBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 7. Invoice Items, Payments, Refunds (sau khi subscriptions và pt_contracts đã có ID)
	itemsBulk := newBulk("invoice_items", "id", "invoice_id", "item_type", "ref_id", "description", "quantity", "unit_price", "discount", "amount")
	itemSeq := uint(1)
	for _, inv := range g.invoices {
		for _, it := range inv.items {
			it.id = itemSeq
			itemSeq++
			var refID *uint
			switch {
			case it.refSub != nil:
				refID = ptrUint(it.refSub.id)
			case it.refContract != nil:
				refID = ptrUint(it.refContract.id)
			case it.refProduct != nil:
				refID = ptrUint(it.refProduct.id)
			}
			itemsBulk.add(it.id, inv.id, it.typ, nUint(refID), it.desc, it.qty, it.unit, it.discount, it.amount)
		}
	}
	if err := itemsBulk.flush(ctx, tx); err != nil {
		return err
	}

	payBulk := newBulk("payments", "id", "invoice_id", "amount", "method", "reference", "paid_at", "received_by", "note", "created_at")
	paySeq := uint(1)
	for _, inv := range g.invoices {
		for _, pm := range inv.payments {
			pm.id = paySeq
			paySeq++
			payBulk.add(pm.id, pm.inv.id, pm.amount, pm.method, pm.reference, pm.at, pm.by.id, pm.note, pm.at)
		}
	}
	if err := payBulk.flush(ctx, tx); err != nil {
		return err
	}

	refBulk := newBulk("refunds", "id", "invoice_id", "amount", "method", "reason", "refunded_by", "created_at")
	refSeq := uint(1)
	for _, inv := range g.invoices {
		for _, rf := range inv.refunds {
			rf.id = refSeq
			refSeq++
			refBulk.add(rf.id, rf.inv.id, rf.amount, rf.method, rf.reason, rf.by.id, rf.at)
		}
	}
	if err := refBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 8. Class Schedules & Bookings
	schBulk := newBulk("class_schedules",
		"id", "class_type_id", "branch_id", "room_id", "trainer_id", "start_at", "end_at",
		"capacity", "booked_count", "status", "series_id", "cancel_reason", "created_at", "updated_at", "created_by",
	)
	bkBulk := newBulk("class_bookings",
		"id", "schedule_id", "member_id", "subscription_id", "status", "waitlist_position",
		"booked_at", "cancelled_at", "attended_at", "source", "created_by", "updated_at",
	)
	bkSeq := uint(1)
	for i, sch := range g.schedules {
		sch.id = uint(i + 1)
		schBulk.add(
			sch.id, sch.ct.id, sch.branch.id, sch.room.id, sch.trainer.id, sch.start, sch.end,
			sch.capacity, sch.booked, sch.status, sch.series, sch.cancelReason, sch.createdAt, sch.createdAt, sch.createdBy.id,
		)
		for _, bk := range sch.bookings {
			bk.id = bkSeq
			bkSeq++
			var subID, cby *uint
			if bk.sub != nil {
				subID = ptrUint(bk.sub.id)
			}
			if bk.createdBy != nil {
				cby = ptrUint(bk.createdBy.id)
			}
			var waitPos any
			if bk.status == entity.BookingStatusWaitlist && bk.waitPos > 0 {
				waitPos = bk.waitPos
			}
			bkBulk.add(
				bk.id, bk.sch.id, bk.m.id, nUint(subID), bk.status, waitPos,
				bk.bookedAt, nTime(bk.cancelledAt), nTime(bk.attendedAt), bk.source, nUint(cby), bk.updatedAt,
			)
		}
	}
	if err := schBulk.flush(ctx, tx); err != nil {
		return err
	}
	if err := bkBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 9. Check-ins & Body Metrics
	chkBulk := newBulk("checkins",
		"id", "member_id", "branch_id", "subscription_id", "checked_in_at", "checked_out_at",
		"method", "result", "deny_reason", "staff_id", "note", "created_at",
	)
	for i, ck := range g.checkins {
		ck.id = uint(i + 1)
		var subID, stID *uint
		if ck.sub != nil {
			subID = ptrUint(ck.sub.id)
		}
		if ck.staff != nil {
			stID = ptrUint(ck.staff.id)
		}
		chkBulk.add(
			ck.id, ck.m.id, ck.branch.id, nUint(subID), ck.in, nTime(ck.out),
			ck.method, ck.result, ck.deny, nUint(stID), ck.note, ck.in,
		)
	}
	if err := chkBulk.flush(ctx, tx); err != nil {
		return err
	}

	metBulk := newBulk("body_metrics",
		"id", "member_id", "measured_at", "weight_kg", "height_cm", "body_fat_pct",
		"muscle_mass_kg", "waist_cm", "chest_cm", "hip_cm", "note", "measured_by", "created_at",
	)
	for i, mt := range g.metrics {
		mt.id = uint(i + 1)
		metBulk.add(
			mt.id, mt.m.id, mt.at, mt.weight, mt.height, nFloat(ptrFloat(mt.fat)),
			nFloat(ptrFloat(mt.muscle)), nFloat(ptrFloat(mt.waist)), nFloat(ptrFloat(mt.chest)),
			nFloat(ptrFloat(mt.hip)), mt.note, mt.by.id, mt.at,
		)
	}
	if err := metBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 10. POS Stock Levels & Stock Transactions
	slBulk := newBulk("stock_levels", "product_id", "branch_id", "quantity", "updated_at")
	for _, p := range g.products {
		if p.catID == 0 {
			continue
		}
		for _, b := range g.branches {
			qty := g.stockBal[[2]uint{p.id, b.id}]
			slBulk.add(p.id, b.id, qty, g.now)
		}
	}
	if err := slBulk.flush(ctx, tx); err != nil {
		return err
	}

	stxBulk := newBulk("stock_transactions",
		"id", "product_id", "branch_id", "type", "quantity", "balance_after", "unit_cost",
		"reference_type", "reference_id", "supplier_id", "voucher_no", "note", "created_by", "created_at",
	)
	for i, stx := range g.stockTxs {
		stx.id = uint(i + 1)
		refType := ""
		var refID *uint
		if stx.inv != nil {
			refType = entity.StockRefInvoice
			refID = ptrUint(stx.inv.id)
		}
		stxBulk.add(
			stx.id, stx.p.id, stx.branch.id, stx.typ, stx.qty, stx.balance, stx.unitCost,
			refType, nUint(refID), nUint(stx.supplierID), stx.voucher, stx.note, stx.by.id, stx.at,
		)
	}
	if err := stxBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 11. Notifications, Audit Logs, Job Runs
	notifBulk := newBulk("notifications", "user_id", "type", "title", "body", "link", "read_at", "created_at")
	for _, nf := range g.notifs {
		uid := nf.userID
		if nf.user != nil {
			uid = nf.user.id
		}
		notifBulk.add(uid, nf.typ, nf.title, nf.body, nf.link, nTime(nf.readAt), nf.at)
	}
	if err := notifBulk.flush(ctx, tx); err != nil {
		return err
	}

	auditBulk := newBulk("audit_logs", "actor_id", "action", "entity", "entity_id", "before", "after", "ip", "created_at")
	for _, au := range g.audits {
		var befJSON, aftJSON any
		if au.before != nil {
			if b, err := json.Marshal(au.before); err == nil {
				befJSON = string(b)
			}
		}
		if au.after != nil {
			if b, err := json.Marshal(au.after()); err == nil {
				aftJSON = string(b)
			}
		}
		eid := au.entityID()
		auditBulk.add(nUint(au.actor), au.action, au.entity, eid, befJSON, aftJSON, au.ip, au.at)
	}
	if err := auditBulk.flush(ctx, tx); err != nil {
		return err
	}

	jobBulk := newBulk("job_runs", "job", "started_at", "finished_at", "status", "affected", "message")
	jStart := atMin(g.today, 5)
	jEnd := jStart.Add(350 * time.Millisecond)
	jobBulk.add("subscriptions.daily", jStart, jEnd, "SUCCESS", 4, "Đã cập nhật trạng thái gói tập và bảo lưu")
	jobBulk.add("classes.no_show", jStart.Add(time.Minute), jEnd.Add(time.Minute), "SUCCESS", 2, "Đã đánh dấu no-show các buổi lớp đã qua")
	if err := jobBulk.flush(ctx, tx); err != nil {
		return err
	}

	// 12. Đồng bộ lại toàn bộ sequence của PostgreSQL sau khi chèn ID tường minh
	return g.syncSequences(ctx, tx)
}

func (g *demoGen) syncSequences(ctx context.Context, tx *gorm.DB) error {
	tables := []string{
		"users", "rooms", "shifts", "members", "body_metrics",
		"membership_plans", "pt_packages", "promotions",
		"invoices", "invoice_items", "payments", "refunds",
		"subscriptions", "subscription_freezes", "subscription_events",
		"checkins", "class_types", "class_schedules", "class_bookings",
		"pt_contracts", "pt_sessions", "trainer_commissions",
		"product_categories", "suppliers", "products", "stock_transactions",
		"notifications", "audit_logs", "job_runs",
	}
	for _, tbl := range tables {
		q := fmt.Sprintf("SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 1), true)", tbl, tbl) // #nosec G201
		if err := tx.WithContext(ctx).Exec(q).Error; err != nil {
			return err
		}
	}
	customSeqs := []struct {
		seq string
		val int
	}{
		{"staff_code_seq", max(1, len(g.staff))},
		{"member_code_seq", max(1, len(g.members))},
		{"invoice_no_seq", max(1, len(g.invoices))},
	}
	for _, cs := range customSeqs {
		if err := tx.WithContext(ctx).Exec("SELECT setval(?, ?, true)", cs.seq, cs.val).Error; err != nil {
			return err
		}
	}
	return nil
}
