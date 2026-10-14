package branch

import (
	"gymmaster-api/api/branch"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	branchService service.BranchService
}

func NewV1(branchService service.BranchService) branch.IBranchV1 {
	return &ControllerV1{branchService: branchService}
}
