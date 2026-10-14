import { useEffect, useState } from 'react';
import { App, Image, Upload } from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import uploadApi from '@/api/upload';
import type { UploadResult } from '@/api/upload';
import { getErrorMessage } from '@/utils/errors';

export interface ImageUploadProps {
  /** URL ảnh (maxCount = 1) hoặc mảng URL (maxCount > 1). */
  value?: string | string[] | null;
  onChange?: (value: string | string[] | undefined) => void;
  maxCount?: number;
  maxSizeMB?: number;
  disabled?: boolean;
  /** Chữ dưới dấu + . */
  buttonText?: string;
}

const ACCEPT = ['image/jpeg', 'image/png', 'image/webp'];

const toArray = (v: ImageUploadProps['value']): string[] =>
  !v ? [] : Array.isArray(v) ? v.filter(Boolean) : [v];

const fileUrl = (f: UploadFile<UploadResult>) => f.url ?? f.response?.url;

const doneUrls = (list: UploadFile<UploadResult>[]) =>
  list
    .filter((f) => f.status === 'done')
    .map(fileUrl)
    .filter((u): u is string => !!u);

/**
 * Tải ảnh dạng picture-card qua `POST /admin/uploads` (field `file`) → lưu URL.
 * Dùng trực tiếp trong Form.Item:
 *   <Form.Item name="avatar_url" label="Ảnh đại diện"><ImageUpload /></Form.Item>
 *   <Form.Item name="images" label="Ảnh"><ImageUpload maxCount={5} /></Form.Item>
 */
export function ImageUpload({
  value,
  onChange,
  maxCount = 1,
  maxSizeMB = 5,
  disabled,
  buttonText = 'Tải ảnh',
}: ImageUploadProps) {
  const { message } = App.useApp();
  const [fileList, setFileList] = useState<UploadFile<UploadResult>[]>([]);
  const [preview, setPreview] = useState<string>();

  // Đồng bộ từ value (form.setFieldsValue) khi không có ảnh đang tải.
  useEffect(() => {
    setFileList((list) => {
      if (list.some((f) => f.status === 'uploading')) return list;
      const urls = toArray(value);
      const current = doneUrls(list);
      if (urls.length === current.length && urls.every((u, i) => u === current[i])) return list;
      return urls.map((url, i) => ({
        uid: `${i}-${url}`,
        name: url.split('/').pop() ?? 'image',
        status: 'done',
        url,
      }));
    });
  }, [value]);

  const emit = (list: UploadFile<UploadResult>[]) => {
    if (list.some((f) => f.status === 'uploading')) return;
    const urls = doneUrls(list);
    onChange?.(maxCount === 1 ? urls[0] : urls);
  };

  const beforeUpload: UploadProps['beforeUpload'] = (file) => {
    if (!ACCEPT.includes(file.type)) {
      message.error('Chỉ nhận ảnh JPG, PNG hoặc WEBP');
      return Upload.LIST_IGNORE;
    }
    if (file.size > maxSizeMB * 1024 * 1024) {
      message.error(`Ảnh tối đa ${maxSizeMB}MB`);
      return Upload.LIST_IGNORE;
    }
    return true;
  };

  const customRequest: UploadProps['customRequest'] = ({
    file,
    onSuccess,
    onError,
    onProgress,
  }) => {
    uploadApi
      .upload(file as File, (percent) => onProgress?.({ percent }))
      .then((res) => onSuccess?.(res))
      .catch((err: Error) => {
        message.error(getErrorMessage(err, 'Tải ảnh thất bại'));
        onError?.(err);
      });
  };

  const handleChange: UploadProps<UploadResult>['onChange'] = ({ fileList: next }) => {
    const list = next
      .filter((f) => f.status !== 'error')
      .map((f) => (f.status === 'done' && !f.url ? { ...f, url: f.response?.url } : f));
    setFileList(list);
    emit(list);
  };

  return (
    <>
      <Upload<UploadResult>
        listType="picture-card"
        accept={ACCEPT.join(',')}
        fileList={fileList}
        maxCount={maxCount}
        multiple={maxCount > 1}
        disabled={disabled}
        beforeUpload={beforeUpload}
        customRequest={customRequest}
        onChange={handleChange}
        onPreview={(f) => setPreview(fileUrl(f) ?? f.thumbUrl)}
      >
        {fileList.length >= maxCount || disabled ? null : (
          <div>
            <PlusOutlined />
            <div style={{ marginTop: 6 }}>{buttonText}</div>
          </div>
        )}
      </Upload>
      {preview && (
        <Image
          wrapperStyle={{ display: 'none' }}
          src={preview}
          preview={{ visible: true, onVisibleChange: (v) => !v && setPreview(undefined) }}
        />
      )}
    </>
  );
}

export default ImageUpload;
