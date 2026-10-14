package seed

import "time"

// Các bản ghi được dựng trong bộ nhớ (tham chiếu bằng con trỏ), gán id theo thứ tự thời gian rồi mới ghi DB.

type dBranch struct {
	id          uint
	idx         int
	code, name  string
	open, close int // phút trong ngày
	manager     *dUser
	receps      []*dUser
	trainers    []*dUser
	rooms       []*dRoom
}

type dUser struct {
	id                 uint
	email, phone, name string
	gender, position   string
	branch             *dBranch
	role               string
	rate               float64
	def                staffDef
	staffCode          string
	hire, dob          time.Time
	salary             float64
	createdAt          time.Time
	lastLogin          *time.Time
	member             *dMember
	busy               []interval // lịch dạy (lớp + PT)
}

type dMember struct {
	id                   uint
	code                 string
	name, gender         string
	dob                  time.Time
	phone, email, idCard string
	address              string
	ecName, ecPhone      string
	branch               *dBranch
	source, status, note string
	createdAt            time.Time
	creator              *dUser
	user                 *dUser
	freq                 float64 // số buổi/tuần
	pref                 int     // 0 sáng, 1 trưa, 2 tối
	classLover           bool
	subs                 []*dSub
	busy                 []interval // PT + lớp đã đặt
	noShows              []time.Time
	demo                 bool
	noRandomEvents       bool
}

type dPlan struct {
	id uint
	planDef
}

type dPTPkg struct {
	id uint
	ptPkgDef
}

type dPromo struct {
	id         uint
	def        promoDef
	start, end time.Time
	used       int
}

type dSub struct {
	id               uint
	m                *dMember
	plan             *dPlan
	branch           *dBranch
	inv              *dInvoice
	start, end       time.Time
	total, remaining int
	price            float64
	status           string
	freezeDays       int
	freezeTimes      int
	prev             *dSub
	soldBy           *dUser
	note             string
	createdAt        time.Time
	updatedAt        time.Time
	activatedAt      *time.Time
	cutoff           *time.Time // từ ngày này gói không còn dùng được (hủy / chuyển nhượng)
	freezes          []*dFreeze
	deniedEmpty      bool
	next             *dSub
	transferredTo    *dSub
}

type dFreeze struct {
	id             uint
	sub            *dSub
	start, end     time.Time
	reason, status string
	requestedBy    *dUser
	reviewedBy     *dUser
	reviewedAt     *time.Time
	reviewNote     string
	createdAt      time.Time
	applied        bool
}

type dEvent struct {
	id    uint
	sub   *dSub
	typ   string
	at    time.Time
	actor *dUser
	data  map[string]any
}

// Giá trị tham chiếu trong data sự kiện, phân giải khi ghi.
type refInvNo struct{ inv *dInvoice }

type dInvoice struct {
	id                                        uint
	no                                        string
	member                                    *dMember
	branch                                    *dBranch
	subtotal, discount, total, paid, refunded float64
	status, source                            string
	promo                                     *dPromo
	note                                      string
	issuedBy                                  *dUser
	issuedAt                                  time.Time
	voidedAt                                  *time.Time
	voidReason                                string
	items                                     []*dItem
	payments                                  []*dPayment
	refunds                                   []*dRefund
	updatedAt                                 time.Time
}

type dItem struct {
	id                     uint
	typ                    string
	refSub                 *dSub
	refContract            *dContract
	refProduct             *dProduct
	desc                   string
	qty                    int
	unit, discount, amount float64
}

type dPayment struct {
	id        uint
	inv       *dInvoice
	amount    float64
	method    string
	reference string
	at        time.Time
	by        *dUser
	note      string
}

type dRefund struct {
	id     uint
	inv    *dInvoice
	amount float64
	method string
	reason string
	by     *dUser
	at     time.Time
}

type dCheckin struct {
	id     uint
	m      *dMember
	branch *dBranch
	sub    *dSub
	in     time.Time
	out    *time.Time
	method string
	result string
	deny   string
	staff  *dUser
	note   string
}

type dRoom struct {
	id       uint
	branch   *dBranch
	name     string
	desc     string
	capacity int
	busy     []interval
}

type dClassType struct {
	id uint
	classTypeDef
}

type dSchedule struct {
	id           uint
	ct           *dClassType
	branch       *dBranch
	room         *dRoom
	trainer      *dUser
	start, end   time.Time
	capacity     int
	booked       int
	status       string
	series       string
	cancelReason string
	createdAt    time.Time
	createdBy    *dUser
	bookings     []*dBooking
	waitPos      int
}

type dBooking struct {
	id          uint
	sch         *dSchedule
	m           *dMember
	sub         *dSub
	status      string
	waitPos     int
	bookedAt    time.Time
	cancelledAt *time.Time
	attendedAt  *time.Time
	source      string
	createdBy   *dUser
	updatedAt   time.Time
}

type dContract struct {
	id          uint
	m           *dMember
	trainer     *dUser
	pkg         *dPTPkg
	branch      *dBranch
	inv         *dInvoice
	total, used int
	minutes     int
	start, end  time.Time
	price       float64
	status      string
	soldBy      *dUser
	note        string
	createdAt   time.Time
	updatedAt   time.Time
	activatedAt time.Time
	sessions    []*dPTSession
}

type dPTSession struct {
	id           uint
	c            *dContract
	start, end   time.Time
	status       string
	workout      string
	tnote        string
	completedAt  *time.Time
	cancelledAt  *time.Time
	cancelReason string
	source       string
	createdBy    *dUser
	createdAt    time.Time
}

type dCommission struct {
	id        uint
	s         *dPTSession
	base      float64
	rate      float64
	amount    float64
	period    string
	status    string
	paidAt    *time.Time
	paidBy    *uint
	createdAt time.Time
}

type dMetric struct {
	id                             uint
	m                              *dMember
	at                             time.Time
	weight, height                 float64
	fat, muscle, waist, chest, hip float64
	note                           string
	by                             *dUser
}

type dProduct struct {
	id       uint
	def      productDef
	cost     float64
	catID    uint
	supplier uint
}

type dStockTx struct {
	id         uint
	p          *dProduct
	branch     *dBranch
	typ        string
	qty        int
	balance    int
	unitCost   float64
	inv        *dInvoice
	supplierID *uint
	voucher    string
	note       string
	by         *dUser
	at         time.Time
}

type dNotif struct {
	user        *dUser
	userID      uint
	typ         string
	title, body string
	link        string
	readAt      *time.Time
	at          time.Time
}

type dAudit struct {
	actor    *uint
	action   string
	entity   string
	entityID func() uint
	before   map[string]any
	after    func() map[string]any
	ip       string
	at       time.Time
}

type dShift struct {
	branch     *dBranch
	staff      *dUser
	start, end time.Time
	note       string
}
