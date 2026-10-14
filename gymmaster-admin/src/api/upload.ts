import { http } from '@/utils/api';

export interface UploadResult {
  url: string;
}

/** Tải ảnh (jpg/png/webp) – `POST /admin/uploads`, multipart field `file`. */
class UploadApi {
  upload(file: File | Blob, onProgress?: (percent: number) => void) {
    const form = new FormData();
    form.append('file', file);
    return http.post<UploadResult>('/admin/uploads', form, {
      onUploadProgress: (e) => {
        if (onProgress && e.total) onProgress(Math.round((e.loaded / e.total) * 100));
      },
    });
  }
}

export const uploadApi = new UploadApi();
export default uploadApi;
