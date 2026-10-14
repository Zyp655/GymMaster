import { useEffect, useRef, useState } from 'react';
import { Alert } from 'antd';
import { getDetectorCtor } from './barcode';

/**
 * Quét QR liên tục bằng camera qua `BarcodeDetector` (không cần thư viện).
 * Gọi `onDetect(text)` mỗi khi đọc được mã (trang tự chống gửi lặp).
 */
export default function CameraScanner({ onDetect }: { onDetect: (text: string) => void }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const cb = useRef(onDetect);
  const [error, setError] = useState<string>();

  useEffect(() => {
    cb.current = onDetect;
  }, [onDetect]);

  useEffect(() => {
    const Ctor = getDetectorCtor();
    if (!Ctor) return;
    let stream: MediaStream | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let stopped = false;
    const detector = new Ctor({ formats: ['qr_code'] });

    const loop = async () => {
      const video = videoRef.current;
      if (stopped || !video) return;
      if (video.readyState >= 2) {
        try {
          const codes = await detector.detect(video);
          const text = codes[0]?.rawValue?.trim();
          if (text) cb.current(text);
        } catch {
          /* khung hình lỗi – bỏ qua */
        }
      }
      timer = setTimeout(() => void loop(), 250);
    };

    navigator.mediaDevices
      .getUserMedia({ video: { facingMode: 'environment' }, audio: false })
      .then((s) => {
        if (stopped) {
          s.getTracks().forEach((t) => t.stop());
          return;
        }
        stream = s;
        if (videoRef.current) {
          videoRef.current.srcObject = s;
          void videoRef.current.play();
        }
        void loop();
      })
      .catch(() =>
        setError('Không mở được camera. Kiểm tra quyền truy cập camera của trình duyệt.'),
      );

    return () => {
      stopped = true;
      clearTimeout(timer);
      stream?.getTracks().forEach((t) => t.stop());
    };
  }, []);

  if (error) return <Alert type="error" showIcon message={error} />;
  return <video ref={videoRef} className="checkin-video" muted playsInline />;
}
