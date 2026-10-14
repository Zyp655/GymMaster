package member

import (
	"gymmaster-api/api/member"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	memberService service.MemberService
}

func NewV1(memberService service.MemberService) member.IMemberV1 {
	return &ControllerV1{memberService: memberService}
}
