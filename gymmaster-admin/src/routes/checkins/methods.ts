import { CHECKIN_METHOD_LABEL, CHECKIN_METHOD_OPTIONS } from '@/api/checkin';

export const METHOD_OPTIONS = CHECKIN_METHOD_OPTIONS;

export const methodLabel = (m: string): string =>
  (CHECKIN_METHOD_LABEL as Record<string, string>)[m] ?? m;
