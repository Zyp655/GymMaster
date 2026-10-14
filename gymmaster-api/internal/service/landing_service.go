package service

import (
	"context"

	v1 "gymmaster-api/api/landing/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/repository"
)

type (
	// LandingService cung cấp dữ liệu công khai cho trang giới thiệu (không cần đăng nhập).
	LandingService interface {
		Plans(ctx context.Context) (*v1.PlansRes, error)
		Branches(ctx context.Context) ([]*v1.PublicBranch, error)
	}

	landingService struct {
		plans    repository.MembershipPlanRepository
		packages repository.PTPackageRepository
		branches repository.BranchRepository
	}
)

func NewLandingService(plans repository.MembershipPlanRepository, packages repository.PTPackageRepository, branches repository.BranchRepository) LandingService {
	return &landingService{plans: plans, packages: packages, branches: branches}
}

func (s *landingService) Plans(ctx context.Context) (*v1.PlansRes, error) {
	plans, err := s.plans.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	pkgs, err := s.packages.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.PlansRes{
		MembershipPlans: dto.Map(plans, func(p entity.MembershipPlan) *v1.PublicPlan {
			return &v1.PublicPlan{
				ID: p.ID, Code: p.Code, Name: p.Name, Description: p.Description, Type: p.Type,
				DurationDays: p.DurationDays, SessionCount: p.SessionCount, Price: p.Price,
				AccessScope: p.AccessScope, MaxFreezeDays: p.MaxFreezeDays, IncludesClasses: p.IncludesClasses,
			}
		}),
		PTPackages: dto.Map(pkgs, func(p entity.PTPackage) *v1.PublicPTPackage {
			return &v1.PublicPTPackage{
				ID: p.ID, Code: p.Code, Name: p.Name, Description: p.Description, SessionCount: p.SessionCount,
				SessionMinutes: p.SessionMinutes, ValidityDays: p.ValidityDays, Price: p.Price,
			}
		}),
	}, nil
}

func (s *landingService) Branches(ctx context.Context) ([]*v1.PublicBranch, error) {
	q := dto.Query{Page: 1, Limit: dto.MaxLimit, OrderBy: "id", OrderDir: "ASC", Filters: map[string]any{"status": consts.StatusActive}}
	items, _, err := s.branches.List(ctx, q)
	if err != nil {
		return nil, err
	}
	return dto.Map(items, func(b entity.Branch) *v1.PublicBranch {
		return &v1.PublicBranch{ID: b.ID, Name: b.Name, Address: b.Address, Phone: b.Phone, OpenTime: b.OpenTime, CloseTime: b.CloseTime}
	}), nil
}
