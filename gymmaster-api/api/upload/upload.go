// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package upload

import (
	"context"

	v1 "gymmaster-api/api/upload/v1"
)

type IUploadV1 interface {
	Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error)
}
