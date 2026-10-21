import { InputNumber } from 'antd';
import type { InputNumberProps } from 'antd';

export type MoneyInputProps = Omit<InputNumberProps<number>, 'formatter' | 'parser'>;

const formatter = (value: number | string | undefined) =>
  value === undefined || value === null || value === ''
    ? ''
    : String(value).replace(/\B(?=(\d{3})+(?!\d))/g, '.');

const parser = (value: string | undefined) => {
  const digits = (value ?? '').replace(/[^\d-]/g, '');
  return digits ? Number(digits) : (undefined as unknown as number);
};

/**
 * Ô nhập tiền VND (phân cách nghìn bằng dấu chấm, hậu tố ₫). Dùng trong Form.Item:
 *   <Form.Item name="price"><MoneyInput /></Form.Item>
 */
export function MoneyInput({ min = 0, step = 1000, style, ...rest }: MoneyInputProps) {
  return (
    <InputNumber<number>
      min={min}
      step={step}
      precision={0}
      suffix="₫"
      formatter={formatter}
      parser={parser}
      style={{ width: '100%', ...style }}
      {...rest}
    />
  );
}

export default MoneyInput;
