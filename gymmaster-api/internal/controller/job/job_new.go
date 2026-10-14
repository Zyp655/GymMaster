package job

import (
	"context"

	"gymmaster-api/api/job"
	"gymmaster-api/internal/service"
)

// EmailTester gửi email thử (service.JobTasks thỏa interface này).
type EmailTester interface {
	SendTestEmail(ctx context.Context, to string) error
}

type ControllerV1 struct {
	jobService service.JobService
	mail       EmailTester
}

func NewV1(jobService service.JobService, mail EmailTester) job.IJobV1 {
	return &ControllerV1{jobService: jobService, mail: mail}
}
