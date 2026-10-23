package seed

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------
// Tên, SĐT, email
// ---------------------------------------------------------------------

func (g *demoGen) personName(gender string) (full, family, middle, given string) {
	family = vnFamilyNames[g.r.weighted(vnFamilyWeights)]
	if gender == "MALE" {
		middle, given = g.r.pickS(vnMaleMiddle), g.r.pickS(vnMaleGiven)
	} else {
		middle, given = g.r.pickS(vnFemaleMiddle), g.r.pickS(vnFemaleGiven)
	}
	if middle == given {
		middle = "Văn"
		if gender != "MALE" {
			middle = "Thị"
		}
	}
	// ~15% tên 4 chữ (thêm đệm thứ hai)
	if g.r.chance(0.15) {
		extra := g.r.pickS(vnFemaleMiddle)
		if gender == "MALE" {
			extra = g.r.pickS(vnMaleMiddle)
		}
		if extra != middle && extra != given {
			middle = middle + " " + extra
		}
	}
	return family + " " + middle + " " + given, family, middle, given
}

func (g *demoGen) uniquePhone() string {
	for {
		p := g.r.pickS(vnPhonePrefixes) + fmt.Sprintf("%07d", g.r.Intn(10000000))
		if !g.phones[p] {
			g.phones[p] = true
			return p
		}
	}
}

func (g *demoGen) reservePhone(p string) string {
	if g.phones[p] {
		return g.uniquePhone()
	}
	g.phones[p] = true
	return p
}

// personalEmail: kiểu tuannv92@gmail.com / nguyenthuha.1998@gmail.com.
func (g *demoGen) personalEmail(family, middle, given string, dob time.Time) string {
	mid := strings.Fields(middle)
	initials := asciiFold(family)[:1]
	for _, m := range mid {
		initials += asciiFold(m)[:1]
	}
	yy := fmt.Sprintf("%02d", dob.Year()%100)
	var local string
	switch g.r.Intn(4) {
	case 0:
		local = asciiFold(given) + initials + yy
	case 1:
		local = asciiFold(family+middle+given) + "." + itoa(dob.Year())
	case 2:
		local = asciiFold(given) + "." + asciiFold(family) + yy
	default:
		local = asciiFold(family+given) + itoa(g.r.between(1, 999))
	}
	dom := emailDomains[g.r.weighted(emailDomainWeights)]
	return g.uniqueEmail(local, dom)
}

func (g *demoGen) uniqueEmail(local, domain string) string {
	e := local + "@" + domain
	for i := 2; g.emails[strings.ToLower(e)]; i++ {
		e = fmt.Sprintf("%s%d@%s", local, i, domain)
	}
	g.emails[strings.ToLower(e)] = true
	return e
}

func (g *demoGen) address(b *dBranch) string {
	streets := hanoiStreets
	city := "Hà Nội"
	if b.idx == 2 {
		streets, city = hcmStreets, "TP.HCM"
	}
	no := g.r.between(1, 320)
	if g.r.chance(0.3) {
		return fmt.Sprintf("Số %d ngõ %d %s, %s", g.r.between(1, 60), no, streets[g.r.Intn(len(streets))], city)
	}
	return fmt.Sprintf("%d %s, %s", no, streets[g.r.Intn(len(streets))], city)
}

func (g *demoGen) idCard(b *dBranch, gender string, dob time.Time) string {
	prov := "001"
	if b.idx == 2 {
		prov = "079"
	}
	c := 0
	if dob.Year() >= 2000 {
		c = 2
	}
	if gender == "FEMALE" {
		c++
	}
	return fmt.Sprintf("%s%d%02d%06d", prov, c, dob.Year()%100, g.r.Intn(1000000))
}

// ---------------------------------------------------------------------
// Nhân sự, phòng, ca làm, loại lớp
// ---------------------------------------------------------------------

func (g *demoGen) genStaff() {
	hireBase := addDays(g.histStart, -400)
	for bi, b := range g.branches {
		tIdx := 0
		for _, def := range demoStaffDefs[bi] {
			u := &dUser{name: def.Name, gender: def.Gender, position: def.Position, branch: b, role: def.Position, rate: def.Rate, def: def}
			_, family, middle, given := splitName(def.Name)
			if def.Email != "" {
				u.email = g.uniqueEmail(strings.Split(def.Email, "@")[0], "gymmaster.vn")
			} else {
				ini := ""
				for _, m := range strings.Fields(family + " " + middle) {
					ini += asciiFold(m)[:1]
				}
				u.email = g.uniqueEmail(asciiFold(given)+"."+ini, "gymmaster.vn")
			}
			u.phone = g.uniquePhone()
			u.dob = time.Date(g.r.between(1985, 2000), time.Month(g.r.between(1, 12)), g.r.between(1, 28), 0, 0, 0, 0, g.loc)
			u.hire = addDays(hireBase, g.r.between(0, 330))
			u.createdAt = atMin(u.hire, g.r.between(8*60, 17*60))
			switch def.Position {
			case "MANAGER":
				u.salary = 18000000
			case "RECEPTIONIST":
				u.salary = float64(g.r.between(75, 90)) * 100000
			default:
				u.salary = float64(g.r.between(80, 120)) * 100000
			}
			ll := g.now.Add(-time.Duration(g.r.between(10, 60*30)) * time.Minute)
			u.lastLogin = &ll
			g.staff = append(g.staff, u)
			switch def.Position {
			case "MANAGER":
				b.manager = u
			case "RECEPTIONIST":
				b.receps = append(b.receps, u)
			case "TRAINER":
				b.trainers = append(b.trainers, u)
				tIdx++
			}
		}
	}
}

func splitName(full string) (string, string, string, string) {
	f := strings.Fields(full)
	return full, f[0], strings.Join(f[1:len(f)-1], " "), f[len(f)-1]
}

func (g *demoGen) genRooms() {
	for bi, b := range g.branches {
		for _, rd := range demoRoomDefs[bi] {
			name := rd.Name
			for g.existingRooms[fmt.Sprintf("%d|%s", b.id, strings.ToLower(name))] {
				name += " (mới)"
			}
			r := &dRoom{branch: b, name: name, desc: rd.Desc, capacity: rd.Capacity}
			b.rooms = append(b.rooms, r)
			g.rooms = append(g.rooms, r)
		}
	}
}

// genShifts: lịch ca tuần này + tuần sau.
// demoShiftHistoryWeeks: số tuần ca làm trong quá khứ (làm nguồn cho chấm công & bảng lương mẫu).
const demoShiftHistoryWeeks = 16

func (g *demoGen) genShifts() {
	monday := addDays(g.today, -(isoWeekday(g.today) - 1))
	for _, b := range g.branches {
		mid := 14 * 60
		if b.idx != 2 {
			mid = 13*60 + 30
		}
		for di := -7 * demoShiftHistoryWeeks; di < 14; di++ {
			d := addDays(monday, di)
			wd := isoWeekday(d)
			k := di + 7*demoShiftHistoryWeeks // chỉ số không âm để xoay vòng lễ tân/HLV
			if wd <= 6 {
				g.shifts = append(g.shifts, &dShift{b, b.manager, atMin(d, 8*60), atMin(d, 17*60), "Ca hành chính"})
			}
			n := len(b.receps)
			morning := b.receps[k%n]
			evening := b.receps[(k+1)%n]
			g.shifts = append(g.shifts, &dShift{b, morning, atMin(d, b.open), atMin(d, mid), "Ca sáng"})
			g.shifts = append(g.shifts, &dShift{b, evening, atMin(d, mid), atMin(d, b.close), "Ca chiều"})
			for ti, t := range b.trainers {
				if (wd+ti)%7 == 0 { // mỗi HLV nghỉ một ngày/tuần
					continue
				}
				if (ti+k/7)%2 == 0 {
					g.shifts = append(g.shifts, &dShift{b, t, atMin(d, b.open), atMin(d, mid), "Ca sáng"})
				} else {
					g.shifts = append(g.shifts, &dShift{b, t, atMin(d, mid), atMin(d, b.close), "Ca chiều"})
				}
			}
		}
	}
}

// ---------------------------------------------------------------------
// Hội viên
// ---------------------------------------------------------------------

const demoMemberCount = 400

func (g *demoGen) genMembers() {
	span := g.today.Sub(g.histStart)
	branchW := []float64{45, 30, 25}
	sources := []string{"WALK_IN", "FACEBOOK", "REFERRAL", "WEBSITE", "OTHER"}
	sourceW := []float64{35, 25, 20, 15, 5}
	for i := 0; i < demoMemberCount; i++ {
		// Mật độ tăng tuyến tính theo thời gian (sqrt) → số hội viên mới tăng dần.
		u := math.Sqrt(g.r.Float64())
		at := g.histStart.Add(time.Duration(u * float64(span)))
		day := dayOf(at, g.loc)
		minute := g.r.between(7*60, 20*60+30)
		created := atMin(day, minute)
		if created.After(g.now) {
			created = g.now.Add(-time.Duration(g.r.between(10, 300)) * time.Minute)
		}
		b := g.branches[g.r.weighted(branchW)]
		m := g.newMember(b, created, sources[g.r.weighted(sourceW)])
		g.members = append(g.members, m)
	}
}

func (g *demoGen) newMember(b *dBranch, created time.Time, source string) *dMember {
	gender := "MALE"
	if g.r.chance(0.45) {
		gender = "FEMALE"
	}
	full, family, middle, given := g.personName(gender)
	age := int(math.Round(g.r.normal(28, 7)))
	if age < 16 {
		age = 16 + g.r.Intn(4)
	}
	if age > 60 {
		age = 60 - g.r.Intn(8)
	}
	dob := time.Date(g.today.Year()-age, time.Month(g.r.between(1, 12)), g.r.between(1, 28), 0, 0, 0, 0, g.loc)
	m := &dMember{name: full, gender: gender, dob: dob, branch: b, source: source, status: "ACTIVE", createdAt: created}
	m.phone = g.uniquePhone()
	if g.r.chance(0.65) {
		m.email = g.personalEmail(family, middle, given, dob)
	}
	if g.r.chance(0.5) {
		m.idCard = g.idCard(b, gender, dob)
	}
	if g.r.chance(0.8) {
		m.address = g.address(b)
	}
	if g.r.chance(0.4) {
		efull, _, _, _ := g.personName([]string{"MALE", "FEMALE"}[g.r.Intn(2)])
		m.ecName, m.ecPhone = efull, g.uniquePhone()
	}
	m.note = g.r.pickS(memberNotes)
	m.creator = g.pickSeller(b)
	m.freq = []float64{1.2, 2, 2.5, 3, 3, 3.5, 4, 4, 5}[g.r.Intn(9)]
	m.pref = g.r.weighted([]float64{40, 15, 45})
	m.classLover = g.r.chance(0.4)
	return m
}

// pickSeller: lễ tân (85%) hoặc quản lý.
func (g *demoGen) pickSeller(b *dBranch) *dUser {
	if g.r.chance(0.15) {
		return b.manager
	}
	return b.receps[g.r.Intn(len(b.receps))]
}

func (g *demoGen) recep(b *dBranch) *dUser { return b.receps[g.r.Intn(len(b.receps))] }

// sortMembers gán mã/thứ tự theo ngày tạo.
func (g *demoGen) sortMembers() {
	sort.SliceStable(g.members, func(i, j int) bool { return g.members[i].createdAt.Before(g.members[j].createdAt) })
}
