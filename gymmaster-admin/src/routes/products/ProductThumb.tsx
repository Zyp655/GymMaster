import { Image } from 'antd';
import { ShoppingOutlined } from '@ant-design/icons';

/** Ảnh nhỏ trong bảng. */
export default function ProductThumb({ src, size = 40 }: { src?: string; size?: number }) {
  return src ? (
    <Image
      src={src}
      width={size}
      height={size}
      style={{ objectFit: 'cover', borderRadius: 6 }}
      preview={{ mask: false }}
    />
  ) : (
    <div
      style={{
        width: size,
        height: size,
        borderRadius: 6,
        background: '#f3f4f6',
        color: '#9ca3af',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <ShoppingOutlined />
    </div>
  );
}

