package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	"gymmaster-api/internal/config"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
)

type demoGen struct {
	r             demoRand
	loc           *time.Location
	now           time.Time
	today         time.Time
	histStart     time.Time
	ownerID       uint
	passHash      string
	roleIDs       map[string]uint
	phones        map[string]bool
	emails        map[string]bool
	existingRooms map[string]bool

	branches   []*dBranch
	plans      []*dPlan
	ptPkgs     []*dPTPkg
	promos     []*dPromo
	classTypes []*dClassType
	catIDs     []uint
	supIDs     []uint
	products   []*dProduct

	staff       []*dUser
	memberUsers []*dUser
	rooms       []*dRoom
	shifts      []*dShift
	members     []*dMember

	subs        []*dSub
	freezes     []*dFreeze
	events      []*dEvent
	invoices    []*dInvoice
	payments    []*dPayment
	refunds     []*dRefund
	schedules   []*dSchedule
	bookings    []*dBooking
	contracts   []*dContract
	ptSessions  []*dPTSession
	commissions []*dCommission
	checkins    []*dCheckin
	metrics     []*dMetric
	stockTxs    []*dStockTx
	stockBal    map[[2]uint]int
	notifs      []*dNotif
	audits      []*dAudit
}

func seedDemo(ctx context.Context, tx *gorm.DB, cfg config.SeedCfg) error {
	var showcaseCount int64
	if err := tx.WithContext(ctx).Model(&entity.User{}).Where("lower(email) = ?", "hoivien@gymmaster.vn").Count(&showcaseCount).Error; err != nil {
		return err
	}
	if showcaseCount > 0 {
		return nil
	}
	var memberCount int64
	if err := tx.WithContext(ctx).Model(&entity.Member{}).Count(&memberCount).Error; err != nil {
		return err
	}
	if memberCount > 0 {
		if err := resetPartialDemo(ctx, tx, cfg.AdminEmail); err != nil {
			return err
		}
	}

	loc := demoLoc()
	now := time.Now().In(loc)
	today := dayOf(now, loc)
	histStart := addDays(today, -150)

	passHash, err := hash.HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}

	gen := &demoGen{
		r:             demoRand{rand.New(rand.NewSource(20261014))}, // #nosec G404
		loc:           loc,
		now:           now,
		today:         today,
		histStart:     histStart,
		passHash:      passHash,
		roleIDs:       make(map[string]uint),
		phones:        make(map[string]bool),
		emails:        make(map[string]bool),
		existingRooms: make(map[string]bool),
		stockBal:      make(map[[2]uint]int),
	}

	if err := gen.loadBase(ctx, tx, cfg.AdminEmail); err != nil {
		return err
	}
	if err := gen.seedCatalog(ctx, tx); err != nil {
		return err
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

	if err := gen.persistAll(ctx, tx); err != nil {
		return err
	}

	g.Log().Infof(ctx, "seed: đã tạo dữ liệu demo (%d nhân sự, %d hội viên, %d gói đăng ký, %d hóa đơn, %d thanh toán, %d hoàn tiền, %d đặt lớp, %d lượt check-in)",
		len(gen.staff), len(gen.members), len(gen.subs), len(gen.invoices), len(gen.payments), len(gen.refunds), len(gen.bookings), len(gen.checkins))
	return nil
}

func resetPartialDemo(ctx context.Context, tx *gorm.DB, ownerEmail string) error {
	q := `TRUNCATE TABLE
		trial_passes, lead_activities, leads,
		maintenance_logs, equipments, reviews,
		workout_logs, workout_plan_items, workout_plan_days, workout_plans,
		online_payment_txns, member_register_otps, password_resets, login_attempts,
		job_runs, audit_logs, notifications,
		stock_transactions, stock_levels, products, suppliers, product_categories,
		trainer_commissions, pt_sessions, pt_contracts, pt_packages,
		class_bookings, class_schedules, class_types,
		checkins, subscription_events, subscription_freezes, subscriptions,
		refunds, payments, invoice_items, invoices, promotions, membership_plans,
		body_metrics, members, shifts, rooms, trainer_profiles, staff_profiles, refresh_tokens
		RESTART IDENTITY CASCADE`
	if err := tx.WithContext(ctx).Exec(q).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec("DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE lower(email) <> lower(?))", ownerEmail).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec("DELETE FROM users WHERE lower(email) <> lower(?)", ownerEmail).Error
}

func (g *demoGen) loadBase(ctx context.Context, tx *gorm.DB, ownerEmail string) error {
	var roles []entity.Role
	if err := tx.WithContext(ctx).Find(&roles).Error; err != nil {
		return err
	}
	for _, r := range roles {
		g.roleIDs[r.Code] = r.ID
	}

	var owner entity.User
	if err := tx.WithContext(ctx).Where("lower(email) = lower(?)", ownerEmail).First(&owner).Error; err != nil {
		return err
	}
	g.ownerID = owner.ID
	g.emails[strings.ToLower(owner.Email)] = true
	if owner.Phone != "" {
		g.phones[owner.Phone] = true
	}

	codes := []string{"HN01", "HN02", "HCM01"}
	for idx, code := range codes {
		var b entity.Branch
		if err := tx.WithContext(ctx).Where("code = ?", code).First(&b).Error; err != nil {
			return err
		}
		g.branches = append(g.branches, &dBranch{
			id:    b.ID,
			idx:   idx,
			code:  b.Code,
			name:  b.Name,
			open:  hhmmToMin(b.OpenTime),
			close: hhmmToMin(b.CloseTime),
		})
	}

	var existingRooms []entity.Room
	if err := tx.WithContext(ctx).Find(&existingRooms).Error; err != nil {
		return err
	}
	for _, rm := range existingRooms {
		g.existingRooms[fmt.Sprintf("%d|%s", rm.BranchID, strings.ToLower(rm.Name))] = true
	}
	return nil
}

func (g *demoGen) seedCatalog(ctx context.Context, tx *gorm.DB) error {
	for _, def := range demoPlanDefs {
		var plan entity.MembershipPlan
		err := tx.WithContext(ctx).Where("code = ?", def.Code).First(&plan).Error
		if err == gorm.ErrRecordNotFound {
			var sess *int
			if def.Type == entity.PlanTypeSession {
				s := def.Sessions
				sess = &s
			}
			plan = entity.MembershipPlan{
				Code:            def.Code,
				Name:            def.Name,
				Description:     def.Desc,
				Type:            def.Type,
				DurationDays:    def.Duration,
				SessionCount:    sess,
				Price:           def.Price,
				AccessScope:     def.Scope,
				MaxFreezeDays:   def.FreezeDays,
				MaxFreezeTimes:  def.FreezeTimes,
				IncludesClasses: def.Classes,
				TransferFee:     def.TransferFee,
				Status:          consts.StatusActive,
				SortOrder:       def.Sort,
			}
			plan.CreatedBy = ptrUint(g.ownerID)
			if err := tx.WithContext(ctx).Create(&plan).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		g.plans = append(g.plans, &dPlan{id: plan.ID, planDef: def})
	}

	for _, def := range demoPTPkgDefs {
		var pkg entity.PTPackage
		err := tx.WithContext(ctx).Where("code = ?", def.Code).First(&pkg).Error
		if err == gorm.ErrRecordNotFound {
			pkg = entity.PTPackage{
				Code:           def.Code,
				Name:           def.Name,
				Description:    def.Desc,
				SessionCount:   def.Sessions,
				SessionMinutes: def.Minutes,
				ValidityDays:   def.Validity,
				Price:          def.Price,
				Status:         consts.StatusActive,
			}
			pkg.CreatedBy = ptrUint(g.ownerID)
			if err := tx.WithContext(ctx).Create(&pkg).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		g.ptPkgs = append(g.ptPkgs, &dPTPkg{id: pkg.ID, ptPkgDef: def})
	}

	for _, def := range demoPromoDefs(g.today.Year()) {
		st, _ := time.ParseInLocation("2006-01-02", def.Start, g.loc)
		en, _ := time.ParseInLocation("2006-01-02", def.End, g.loc)
		en = en.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		var promo entity.Promotion
		err := tx.WithContext(ctx).Where("upper(code) = upper(?)", def.Code).First(&promo).Error
		if err == gorm.ErrRecordNotFound {
			var maxD *float64
			if def.MaxDiscount > 0 {
				maxD = ptrFloat(def.MaxDiscount)
			}
			var lim *int
			if def.Limit > 0 {
				l := def.Limit
				lim = &l
			}
			promo = entity.Promotion{
				Code:        def.Code,
				Name:        def.Name,
				Type:        def.Type,
				Value:       def.Value,
				MaxDiscount: maxD,
				MinAmount:   def.MinAmt,
				AppliesTo:   def.AppliesTo,
				StartAt:     st,
				EndAt:       en,
				UsageLimit:  lim,
				Status:      consts.StatusActive,
			}
			promo.CreatedBy = ptrUint(g.ownerID)
			if err := tx.WithContext(ctx).Create(&promo).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		g.promos = append(g.promos, &dPromo{id: promo.ID, def: def, start: st, end: en})
	}

	for _, def := range demoClassTypeDefs {
		var ct entity.ClassType
		err := tx.WithContext(ctx).Where("code = ?", def.Code).First(&ct).Error
		if err == gorm.ErrRecordNotFound {
			ct = entity.ClassType{
				Code:               def.Code,
				Name:               def.Name,
				Description:        def.Desc,
				Color:              def.Color,
				DefaultDurationMin: def.Duration,
				DefaultCapacity:    def.Capacity,
				Level:              def.Level,
				Status:             consts.StatusActive,
			}
			ct.CreatedBy = ptrUint(g.ownerID)
			if err := tx.WithContext(ctx).Create(&ct).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		g.classTypes = append(g.classTypes, &dClassType{id: ct.ID, classTypeDef: def})
	}

	for _, def := range demoCategoryDefs {
		var cat entity.ProductCategory
		err := tx.WithContext(ctx).Where("lower(name) = lower(?)", def.Name).First(&cat).Error
		if err == gorm.ErrRecordNotFound {
			cat = entity.ProductCategory{Name: def.Name, SortOrder: def.Sort}
			if err := tx.WithContext(ctx).Create(&cat).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		g.catIDs = append(g.catIDs, cat.ID)
	}

	for _, def := range demoSupplierDefs {
		var sup entity.Supplier
		err := tx.WithContext(ctx).Where("lower(name) = lower(?)", def.Name).First(&sup).Error
		if err == gorm.ErrRecordNotFound {
			sup = entity.Supplier{
				Name:    def.Name,
				Phone:   def.Phone,
				Email:   def.Email,
				Address: def.Address,
				Note:    def.Note,
			}
			if err := tx.WithContext(ctx).Create(&sup).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		g.supIDs = append(g.supIDs, sup.ID)
	}

	for _, def := range demoProductDefs {
		var prod entity.Product
		err := tx.WithContext(ctx).Where("sku = ?", def.SKU).First(&prod).Error
		catID := g.catIDs[def.Category]
		supID := g.supIDs[def.Supplier]
		if err == gorm.ErrRecordNotFound {
			var memPrice *float64
			if def.Member > 0 {
				memPrice = ptrFloat(def.Member)
			}
			prod = entity.Product{
				SKU:         def.SKU,
				Barcode:     def.Barcode,
				Name:        def.Name,
				CategoryID:  ptrUint(catID),
				Unit:        def.Unit,
				CostPrice:   def.Cost,
				SellPrice:   def.Sell,
				MemberPrice: memPrice,
				MinStock:    def.MinStock,
				Status:      entity.ProductStatusActive,
			}
			prod.CreatedBy = ptrUint(g.ownerID)
			if err := tx.WithContext(ctx).Create(&prod).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		g.products = append(g.products, &dProduct{
			id:       prod.ID,
			def:      def,
			cost:     def.Cost,
			catID:    catID,
			supplier: supID,
		})
	}

	return nil
}

func (g *demoGen) addShowcaseMember() {
	b := g.branches[0]
	created := atMin(addDays(g.today, -95), 9*60+15)
	dob := time.Date(1996, time.May, 18, 0, 0, 0, 0, g.loc)
	m := &dMember{
		name:           "Nguyễn Minh Quân",
		gender:         "MALE",
		dob:            dob,
		phone:          g.reservePhone("0912345678"),
		email:          g.uniqueEmail("hoivien", "gymmaster.vn"),
		idCard:         g.idCard(b, "MALE", dob),
		address:        "144 Xuân Thủy, Cầu Giấy, Hà Nội",
		ecName:         "Nguyễn Thị Thu Hà",
		ecPhone:        g.reservePhone("0987654321"),
		branch:         b,
		source:         "WEBSITE",
		status:         "ACTIVE",
		note:           "Tài khoản hội viên mẫu để trải nghiệm Cổng hội viên (/me).",
		createdAt:      created,
		creator:        b.receps[0],
		freq:           4,
		pref:           2,
		classLover:     true,
		demo:           true,
		noRandomEvents: true,
	}
	g.members = append(g.members, m)
}

func (g *demoGen) assignMemberUsers() {
	for i, m := range g.members {
		if !m.demo && (m.email == "" || i%10 != 0) {
			continue
		}
		ll := g.now.Add(-time.Duration(g.r.between(30, 60*48)) * time.Minute)
		u := &dUser{
			email:     m.email,
			phone:     m.phone,
			name:      m.name,
			gender:    m.gender,
			position:  "OTHER",
			branch:    m.branch,
			role:      consts.RoleMember,
			dob:       m.dob,
			createdAt: m.createdAt.Add(5 * time.Minute),
			lastLogin: &ll,
			member:    m,
		}
		m.user = u
		g.memberUsers = append(g.memberUsers, u)
	}
}

func (g *demoGen) findPromo(at time.Time, subtotal float64) *dPromo {
	for _, p := range g.promos {
		if (at.Equal(p.start) || at.After(p.start)) && (at.Equal(p.end) || at.Before(p.end)) {
			if subtotal < p.def.MinAmt {
				continue
			}
			if p.def.Limit > 0 && p.used >= p.def.Limit {
				continue
			}
			return p
		}
	}
	return nil
}

func (g *demoGen) calcPromoDiscount(p *dPromo, subtotal float64) float64 {
	if p == nil {
		return 0
	}
	var disc float64
	if p.def.Type == entity.PromotionPercent {
		disc = roundTo(subtotal*p.def.Value/100, 1000)
		if p.def.MaxDiscount > 0 && disc > p.def.MaxDiscount {
			disc = p.def.MaxDiscount
		}
	} else {
		disc = p.def.Value
	}
	if disc > subtotal {
		disc = subtotal
	}
	return roundVND(disc)
}

func (g *demoGen) createSubSale(m *dMember, plan *dPlan, startDay time.Time, issuedAt time.Time, seller *dUser, prev *dSub, evType string, priceOverride *float64) *dSub {
	subtotal := plan.Price
	if priceOverride != nil {
		subtotal = roundVND(*priceOverride)
	}
	var promo *dPromo
	var discount float64
	if priceOverride == nil && g.r.chance(0.25) {
		promo = g.findPromo(issuedAt, subtotal)
		discount = g.calcPromoDiscount(promo, subtotal)
		if promo != nil && discount > 0 {
			promo.used++
		} else {
			promo = nil
		}
	}
	total := roundVND(subtotal - discount)

	inv := &dInvoice{
		member:    m,
		branch:    m.branch,
		subtotal:  subtotal,
		discount:  discount,
		total:     total,
		paid:      total,
		status:    entity.InvoiceStatusPaid,
		source:    entity.InvoiceSourceCounter,
		promo:     promo,
		note:      fmt.Sprintf("Đăng ký %s (%s)", plan.Name, moneyVN(total)),
		issuedBy:  seller,
		issuedAt:  issuedAt,
		updatedAt: issuedAt,
	}

	endDay := addDays(startDay, plan.Duration-1)
	totSess := 0
	remSess := 0
	if plan.Type == entity.PlanTypeSession {
		totSess = plan.Sessions
		remSess = plan.Sessions
	}

	actAt := issuedAt
	sub := &dSub{
		m:           m,
		plan:        plan,
		branch:      m.branch,
		inv:         inv,
		start:       startDay,
		end:         endDay,
		total:       totSess,
		remaining:   remSess,
		price:       total,
		status:      entity.SubStatusActive,
		prev:        prev,
		soldBy:      seller,
		note:        plan.Name,
		createdAt:   issuedAt,
		updatedAt:   issuedAt,
		activatedAt: &actAt,
	}
	if prev != nil {
		prev.next = sub
	}

	item := &dItem{
		typ:      entity.InvoiceItemMembership,
		refSub:   sub,
		desc:     fmt.Sprintf("%s (%s – %s)", plan.Name, dmy(startDay), dmy(endDay)),
		qty:      1,
		unit:     subtotal,
		discount: discount,
		amount:   total,
	}
	inv.items = append(inv.items, item)

	method := g.r.pickS([]string{entity.PaymentTransfer, entity.PaymentCash, entity.PaymentCard})
	ref := ""
	if method != entity.PaymentCash {
		ref = fmt.Sprintf("FT%s%04d", issuedAt.Format("060102"), g.r.between(1000, 9999))
	}
	pm := &dPayment{
		inv:       inv,
		amount:    total,
		method:    method,
		reference: ref,
		at:        issuedAt.Add(time.Minute),
		by:        seller,
		note:      "Thu đủ tại quầy",
	}
	inv.payments = append(inv.payments, pm)

	g.invoices = append(g.invoices, inv)
	g.payments = append(g.payments, pm)
	g.subs = append(g.subs, sub)
	m.subs = append(m.subs, sub)

	g.events = append(g.events, &dEvent{
		sub:   sub,
		typ:   evType,
		at:    issuedAt,
		actor: seller,
		data: map[string]any{
			"plan_code":  plan.Code,
			"invoice_no": refInvNo{inv: inv},
			"amount":     total,
		},
	})
	g.events = append(g.events, &dEvent{
		sub:   sub,
		typ:   entity.SubEventActivated,
		at:    issuedAt.Add(time.Minute),
		actor: seller,
		data: map[string]any{
			"start_date": ymd(startDay),
			"end_date":   ymd(endDay),
		},
	})

	return sub
}

func (g *demoGen) genSubscriptions() {
	planWeights := []float64{32, 28, 15, 12, 9, 4}

	for idx, m := range g.members {
		seller := g.pickSeller(m.branch)
		firstStart := dayOf(m.createdAt, g.loc)

		if m.demo {
			// Gói 1: Gói 1 tháng đã hết hạn, sau đó gia hạn lên Gói 3 tháng đang ACTIVE
			p1 := g.plans[0] // M1
			s1 := g.createSubSale(m, p1, firstStart, m.createdAt, seller, nil, entity.SubEventCreated, nil)
			s1.status = entity.SubStatusExpired
			s1.updatedAt = atMin(addDays(s1.end, 1), 10)
			g.events = append(g.events, &dEvent{
				sub:  s1,
				typ:  entity.SubEventExpired,
				at:   s1.updatedAt,
				data: map[string]any{"end_date": ymd(s1.end)},
			})

			p2 := g.plans[1] // M3 (ALL_BRANCHES, includes classes)
			renewStart := addDays(s1.end, 1)
			renewAt := atMin(s1.end, 18*60+15)
			s2 := g.createSubSale(m, p2, renewStart, renewAt, seller, s1, entity.SubEventRenewed, nil)

			// Tạo 1 bảo lưu đã kết thúc cho gói hiện tại
			fzStart := addDays(g.today, -25)
			fzEnd := addDays(g.today, -21)
			fzDays := daysBetween(fzStart, fzEnd) + 1
			revAt := atMin(addDays(fzStart, -1), 10*60)
			fz := &dFreeze{
				sub:         s2,
				start:       fzStart,
				end:         fzEnd,
				reason:      "Đi công tác Đà Nẵng",
				status:      entity.FreezeStatusEnded,
				requestedBy: m.user,
				reviewedBy:  m.branch.manager,
				reviewedAt:  ptrTime(revAt),
				reviewNote:  "Đã duyệt bảo lưu",
				createdAt:   revAt.Add(-2 * time.Hour),
				applied:     true,
			}
			s2.freezes = append(s2.freezes, fz)
			s2.freezeDays += fzDays
			s2.freezeTimes++
			s2.end = addDays(s2.end, fzDays)
			g.freezes = append(g.freezes, fz)
			g.events = append(g.events,
				&dEvent{sub: s2, typ: entity.SubEventFreezeRequested, at: fz.createdAt, actor: m.user, data: map[string]any{"days": fzDays}},
				&dEvent{sub: s2, typ: entity.SubEventFrozen, at: revAt, actor: m.branch.manager, data: map[string]any{"from": ymd(fzStart), "to": ymd(fzEnd)}},
				&dEvent{sub: s2, typ: entity.SubEventUnfrozen, at: atMin(addDays(fzEnd, 1), 15), data: map[string]any{"new_end_date": ymd(s2.end)}},
			)
			continue
		}

		plan := g.plans[g.r.weighted(planWeights)]
		cur := g.createSubSale(m, plan, firstStart, m.createdAt, seller, nil, entity.SubEventCreated, nil)

		// Chuỗi gia hạn / nâng cấp nếu gói kết thúc trước hôm nay
		for cur.end.Before(g.today) {
			// 12% nâng cấp khi còn vài ngày, hoặc 62% gia hạn sau khi hết hạn
			if cur.plan.Duration <= 90 && g.r.chance(0.10) {
				upDay := addDays(cur.end, -g.r.between(3, 10))
				if upDay.After(cur.start) && upDay.Before(g.today) {
					upAt := atMin(upDay, g.r.between(9*60, 19*60))
					newPlan := g.plans[3] // M12
					remDays := max(1, daysBetween(upDay, cur.end))
					credit := roundTo(cur.price*float64(remDays)/float64(cur.plan.Duration), 1000)
					diffPrice := math.Max(100000, newPlan.Price-credit)
					cur.status = entity.SubStatusCancelled
					cur.cutoff = ptrTime(upDay)
					cur.updatedAt = upAt
					nextSub := g.createSubSale(m, newPlan, upDay, upAt, g.pickSeller(m.branch), cur, entity.SubEventUpgraded, &diffPrice)
					cur = nextSub
					continue
				}
			}

			// Đánh dấu hết hạn gói cũ
			expAt := atMin(addDays(cur.end, 1), 5)
			cur.status = entity.SubStatusExpired
			cur.updatedAt = expAt
			g.events = append(g.events, &dEvent{
				sub:  cur,
				typ:  entity.SubEventExpired,
				at:   expAt,
				data: map[string]any{"end_date": ymd(cur.end)},
			})

			if !g.r.chance(0.62) {
				break
			}
			gap := g.r.between(0, 7)
			nextStart := addDays(cur.end, 1+gap)
			if nextStart.After(g.today) {
				nextStart = g.today
			}
			renewAt := atMin(nextStart, g.r.between(8*60, 20*60))
			if renewAt.After(g.now) {
				renewAt = g.now.Add(-time.Hour)
			}
			nextPlan := g.plans[g.r.weighted(planWeights)]
			cur = g.createSubSale(m, nextPlan, nextStart, renewAt, g.pickSeller(m.branch), cur, entity.SubEventRenewed, nil)
		}

		// Một số gói đang chạy có bảo lưu (ENDED, FROZEN, PENDING, REJECTED)
		if !m.noRandomEvents && cur.status == entity.SubStatusActive && cur.plan.FreezeTimes > 0 {
			if idx%28 == 0 && daysBetween(cur.start, g.today) > 20 && daysBetween(g.today, cur.end) > 15 {
				// Đang bảo lưu (FROZEN)
				fzStart := addDays(g.today, -3)
				fzEnd := addDays(g.today, 4)
				fzDays := daysBetween(fzStart, fzEnd) + 1
				revAt := atMin(fzStart, 9*60)
				fz := &dFreeze{
					sub:         cur,
					start:       fzStart,
					end:         fzEnd,
					reason:      g.r.pickS(freezeReasons),
					status:      entity.FreezeStatusApproved,
					requestedBy: seller,
					reviewedBy:  m.branch.manager,
					reviewedAt:  ptrTime(revAt),
					reviewNote:  "Đồng ý bảo lưu",
					createdAt:   revAt.Add(-time.Hour),
					applied:     true,
				}
				cur.status = entity.SubStatusFrozen
				cur.freezeDays += fzDays
				cur.freezeTimes++
				cur.end = addDays(cur.end, fzDays)
				cur.freezes = append(cur.freezes, fz)
				g.freezes = append(g.freezes, fz)
				g.events = append(g.events,
					&dEvent{sub: cur, typ: entity.SubEventFreezeRequested, at: fz.createdAt, actor: seller, data: map[string]any{"days": fzDays}},
					&dEvent{sub: cur, typ: entity.SubEventFrozen, at: revAt, actor: m.branch.manager, data: map[string]any{"from": ymd(fzStart), "to": ymd(fzEnd)}},
				)
			} else if idx%35 == 0 && daysBetween(g.today, cur.end) > 15 {
				// Yêu cầu bảo lưu chờ duyệt (PENDING)
				fzStart := addDays(g.today, 2)
				fzEnd := addDays(g.today, 8)
				fz := &dFreeze{
					sub:         cur,
					start:       fzStart,
					end:         fzEnd,
					reason:      g.r.pickS(freezeReasons),
					status:      entity.FreezeStatusPending,
					requestedBy: seller,
					createdAt:   g.now.Add(-time.Duration(g.r.between(30, 360)) * time.Minute),
				}
				cur.freezes = append(cur.freezes, fz)
				g.freezes = append(g.freezes, fz)
				g.events = append(g.events, &dEvent{
					sub:   cur,
					typ:   entity.SubEventFreezeRequested,
					at:    fz.createdAt,
					actor: seller,
					data:  map[string]any{"from": ymd(fzStart), "to": ymd(fzEnd)},
				})
			} else if idx%45 == 0 && daysBetween(cur.start, g.today) > 10 {
				// Yêu cầu bảo lưu bị từ chối (REJECTED)
				fzStart := addDays(g.today, -5)
				fzEnd := addDays(g.today, -1)
				revAt := atMin(fzStart, 11*60)
				fz := &dFreeze{
					sub:         cur,
					start:       fzStart,
					end:         fzEnd,
					reason:      g.r.pickS(freezeReasons),
					status:      entity.FreezeStatusRejected,
					requestedBy: seller,
					reviewedBy:  m.branch.manager,
					reviewedAt:  ptrTime(revAt),
					reviewNote:  "Không đủ điều kiện chứng từ bảo lưu",
					createdAt:   revAt.Add(-2 * time.Hour),
				}
				cur.freezes = append(cur.freezes, fz)
				g.freezes = append(g.freezes, fz)
				g.events = append(g.events, &dEvent{
					sub:   cur,
					typ:   entity.SubEventFreezeRejected,
					at:    revAt,
					actor: m.branch.manager,
					data:  map[string]any{"reason": fz.reviewNote},
				})
			}
		}

		// Một vài gói bị hủy hoàn tiền một phần (CANCELLED + REFUND)
		if !m.noRandomEvents && idx%55 == 0 && cur.status == entity.SubStatusActive && daysBetween(cur.start, g.today) > 10 {
			cancelDay := addDays(g.today, -2)
			cancelAt := atMin(cancelDay, 15*60)
			cur.status = entity.SubStatusCancelled
			cur.cutoff = ptrTime(cancelDay)
			cur.note = g.r.pickS(cancelReasons)
			cur.updatedAt = cancelAt

			refAmt := roundTo(cur.price*0.4, 1000)
			if refAmt > 0 && cur.inv != nil {
				rf := &dRefund{
					inv:    cur.inv,
					amount: refAmt,
					method: entity.PaymentTransfer,
					reason: cur.note,
					by:     m.branch.manager,
					at:     cancelAt,
				}
				cur.inv.refunded += refAmt
				cur.inv.refunds = append(cur.inv.refunds, rf)
				cur.inv.updatedAt = cancelAt
				g.refunds = append(g.refunds, rf)
			}
			g.events = append(g.events, &dEvent{
				sub:   cur,
				typ:   entity.SubEventCancelled,
				at:    cancelAt,
				actor: m.branch.manager,
				data:  map[string]any{"reason": cur.note, "refund": refAmt},
			})
		}

		// Một vài hội viên không còn gói active chuyển trạng thái INACTIVE
		if cur.status == entity.SubStatusExpired && daysBetween(cur.end, g.today) > 45 && g.r.chance(0.3) {
			m.status = entity.MemberStatusInactive
		}
	}

	// Thực hiện 2 ca chuyển nhượng gói tập (TRANSFERRED + hóa đơn FEE)
	g.genTransfers()
	// Tạo 2 hóa đơn VOID và 2 hóa đơn PARTIAL để đầy đủ trạng thái tài chính
	g.genSpecialInvoices()
}

func (g *demoGen) genTransfers() {
	var donors []*dSub
	var receivers []*dMember
	for _, m := range g.members {
		if m.demo || m.noRandomEvents || len(m.subs) == 0 {
			continue
		}
		last := m.subs[len(m.subs)-1]
		if last.status == entity.SubStatusActive && last.plan.TransferFee > 0 && daysBetween(last.start, g.today) > 15 && daysBetween(g.today, last.end) > 30 {
			if len(donors) < 2 {
				donors = append(donors, last)
				continue
			}
		}
		if last.status == entity.SubStatusExpired && len(donors) > 0 && len(receivers) < 2 && m.branch == donors[0].branch {
			receivers = append(receivers, m)
		}
	}
	for i := 0; i < len(donors) && i < len(receivers); i++ {
		oldSub := donors[i]
		target := receivers[i]
		tfDay := addDays(g.today, -5-i*3)
		tfAt := atMin(tfDay, 14*60)
		seller := g.recep(oldSub.branch)

		oldSub.status = entity.SubStatusTransferred
		oldSub.cutoff = ptrTime(tfDay)
		oldSub.updatedAt = tfAt

		fee := oldSub.plan.TransferFee
		feeInv := &dInvoice{
			member:    target,
			branch:    oldSub.branch,
			subtotal:  fee,
			total:     fee,
			paid:      fee,
			status:    entity.InvoiceStatusPaid,
			source:    entity.InvoiceSourceCounter,
			note:      fmt.Sprintf("Phí chuyển nhượng gói %s", oldSub.plan.Code),
			issuedBy:  seller,
			issuedAt:  tfAt,
			updatedAt: tfAt,
		}

		newSub := &dSub{
			m:           target,
			plan:        oldSub.plan,
			branch:      oldSub.branch,
			inv:         feeInv,
			start:       tfDay,
			end:         oldSub.end,
			total:       oldSub.total,
			remaining:   oldSub.remaining,
			price:       fee,
			status:      entity.SubStatusActive,
			prev:        oldSub,
			soldBy:      seller,
			note:        fmt.Sprintf("Nhận chuyển nhượng từ %s", oldSub.m.name),
			createdAt:   tfAt,
			updatedAt:   tfAt,
			activatedAt: ptrTime(tfAt),
		}
		oldSub.transferredTo = newSub
		target.subs = append(target.subs, newSub)
		target.status = entity.MemberStatusActive
		g.subs = append(g.subs, newSub)

		feeItem := &dItem{
			typ:    entity.InvoiceItemFee,
			refSub: newSub,
			desc:   fmt.Sprintf("Phí chuyển nhượng gói %s", oldSub.plan.Name),
			qty:    1,
			unit:   fee,
			amount: fee,
		}
		feeInv.items = append(feeInv.items, feeItem)
		pm := &dPayment{
			inv:    feeInv,
			amount: fee,
			method: entity.PaymentCash,
			at:     tfAt.Add(time.Minute),
			by:     seller,
			note:   "Thu phí chuyển nhượng",
		}
		feeInv.payments = append(feeInv.payments, pm)
		g.invoices = append(g.invoices, feeInv)
		g.payments = append(g.payments, pm)

		g.events = append(g.events,
			&dEvent{sub: oldSub, typ: entity.SubEventTransferred, at: tfAt, actor: seller, data: map[string]any{"to_member": target.name}},
			&dEvent{sub: newSub, typ: entity.SubEventCreated, at: tfAt, actor: seller, data: map[string]any{"from_member": oldSub.m.name}},
		)
	}
}

func (g *demoGen) genSpecialInvoices() {
	if len(g.members) < 10 {
		return
	}
	// 1 hóa đơn VOID (nhập sai gói)
	m1 := g.members[5]
	b1 := m1.branch
	at1 := atMin(addDays(g.today, -4), 11*60)
	vAt := at1.Add(15 * time.Minute)
	invVoid := &dInvoice{
		member:     m1,
		branch:     b1,
		subtotal:   500000,
		total:      500000,
		status:     entity.InvoiceStatusVoid,
		source:     entity.InvoiceSourceCounter,
		note:       "Lập nhầm gói tập",
		issuedBy:   b1.receps[0],
		issuedAt:   at1,
		voidedAt:   &vAt,
		voidReason: "Lễ tân chọn nhầm gói 1 tháng thay vì 3 tháng",
		updatedAt:  vAt,
	}
	invVoid.items = append(invVoid.items, &dItem{
		typ:    entity.InvoiceItemOther,
		desc:   "Gói 1 tháng (nhập nhầm)",
		qty:    1,
		unit:   500000,
		amount: 500000,
	})
	g.invoices = append(g.invoices, invVoid)

	// 1 hóa đơn PARTIAL (thanh toán một phần)
	m2 := g.members[8]
	b2 := m2.branch
	at2 := atMin(addDays(g.today, -1), 16*60)
	invPart := &dInvoice{
		member:    m2,
		branch:    b2,
		subtotal:  4800000,
		total:     4800000,
		paid:      2000000,
		status:    entity.InvoiceStatusPartial,
		source:    entity.InvoiceSourceCounter,
		note:      "Đặt cọc gói 12 tháng, hẹn thanh toán phần còn lại",
		issuedBy:  b2.receps[0],
		issuedAt:  at2,
		updatedAt: at2,
	}
	invPart.items = append(invPart.items, &dItem{
		typ:    entity.InvoiceItemOther,
		desc:   "Đặt cọc Gói 12 tháng",
		qty:    1,
		unit:   4800000,
		amount: 4800000,
	})
	pm := &dPayment{
		inv:    invPart,
		amount: 2000000,
		method: entity.PaymentTransfer,
		at:     at2.Add(2 * time.Minute),
		by:     b2.receps[0],
		note:   "Thu tiền cọc đợt 1",
	}
	invPart.payments = append(invPart.payments, pm)
	g.invoices = append(g.invoices, invPart)
	g.payments = append(g.payments, pm)
}

func resolveEventData(data map[string]any) []byte {
	if len(data) == 0 {
		return []byte("{}")
	}
	out := make(map[string]any, len(data))
	for k, v := range data {
		if r, ok := v.(refInvNo); ok && r.inv != nil {
			out[k] = r.inv.no
		} else {
			out[k] = v
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func sortInvoicesChronologically(list []*dInvoice) {
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].issuedAt.Before(list[j].issuedAt)
	})
}
