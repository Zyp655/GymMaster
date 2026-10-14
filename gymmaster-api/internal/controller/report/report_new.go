package report

import (
	"gymmaster-api/api/report"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	reportService service.ReportService
}

func NewV1(reportService service.ReportService) report.IReportV1 {
	return &ControllerV1{reportService: reportService}
}
