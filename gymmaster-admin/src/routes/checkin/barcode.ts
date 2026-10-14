/** Barcode Detection API của trình duyệt (quét QR không cần thư viện). */
/** Kiểu tối thiểu của Barcode Detection API (chưa có trong lib.dom của TS). */
interface DetectedBarcode {
  rawValue: string;
}
interface BarcodeDetectorLike {
  detect(source: CanvasImageSource): Promise<DetectedBarcode[]>;
}
type BarcodeDetectorCtor = new (opts?: { formats?: string[] }) => BarcodeDetectorLike;

export function getDetectorCtor(): BarcodeDetectorCtor | undefined {
  return (window as unknown as { BarcodeDetector?: BarcodeDetectorCtor }).BarcodeDetector;
}

/** Trình duyệt hỗ trợ quét QR bằng camera (BarcodeDetector + getUserMedia). */
export const cameraScanSupported = () =>
  typeof window !== 'undefined' && !!getDetectorCtor() && !!navigator.mediaDevices?.getUserMedia;
