import { useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties, MouseEvent } from 'react';
import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { ClassSchedule } from '@/api/class';
import { readableTextColor, weekdayName, withAlpha } from './constants';

const HOUR_PX = 64;
const MIN_PX = HOUR_PX / 60;

export interface WeekCalendarProps {
  /** Thứ Hai của tuần đang xem. */
  weekStart: Dayjs;
  schedules: ClassSchedule[];
  /** Giờ mở/đóng cửa `HH:mm` – trục giờ (tự nới nếu có lớp nằm ngoài). */
  openTime?: string;
  closeTime?: string;
  onSelect: (s: ClassSchedule) => void;
  /** Bấm ô trống → tạo lịch tại thời điểm đó (làm tròn 30 phút). */
  onCreateAt?: (at: Dayjs) => void;
}

const toMin = (hhmm: string | undefined, fallback: number) => {
  const m = /^(\d{1,2}):(\d{2})/.exec(hhmm ?? '');
  return m ? Number(m[1]) * 60 + Number(m[2]) : fallback;
};

const minuteOfDay = (d: Dayjs) => d.hour() * 60 + d.minute();

interface Placed {
  s: ClassSchedule;
  start: number;
  end: number;
  lane: number;
  lanes: number;
}

/** Xếp các buổi chồng giờ trong một ngày thành các làn song song. */
function layoutDay(items: ClassSchedule[]): Placed[] {
  const sorted = items
    .map((s) => {
      const start = minuteOfDay(dayjs(s.start_at));
      return { s, start, end: start + Math.max(s.duration_min, 15), lane: 0, lanes: 1 };
    })
    .sort((a, b) => a.start - b.start || b.end - a.end);

  const result: Placed[] = [];
  let cluster: Placed[] = [];
  let laneEnds: number[] = [];
  let clusterEnd = -1;

  const flush = () => {
    cluster.forEach((p) => (p.lanes = laneEnds.length));
    result.push(...cluster);
    cluster = [];
    laneEnds = [];
  };

  sorted.forEach((p) => {
    if (cluster.length && p.start >= clusterEnd) flush();
    let lane = laneEnds.findIndex((end) => end <= p.start);
    if (lane === -1) {
      lane = laneEnds.length;
      laneEnds.push(p.end);
    } else laneEnds[lane] = p.end;
    p.lane = lane;
    cluster.push(p);
    clusterEnd = Math.max(clusterEnd, p.end);
  });
  flush();
  return result;
}

function useNow(intervalMs = 60_000) {
  const [now, setNow] = useState(() => dayjs());
  useEffect(() => {
    const t = window.setInterval(() => setNow(dayjs()), intervalMs);
    return () => window.clearInterval(t);
  }, [intervalMs]);
  return now;
}

/** Nội dung tóm tắt một buổi (tooltip). */
export function ScheduleSummary({ s }: { s: ClassSchedule }) {
  const start = dayjs(s.start_at);
  const end = dayjs(s.end_at);
  return (
    <div style={{ fontSize: 12, lineHeight: 1.5 }}>
      <b>{s.class_type_name}</b>
      <div>
        {start.format('HH:mm')}–{end.format('HH:mm')} · {weekdayName(start)} {start.format('DD/MM')}
      </div>
      <div>HLV: {s.trainer_name}</div>
      <div>Phòng: {s.room_name}</div>
      <div>
        Đã đặt: {s.booked_count}/{s.capacity}
        {s.waitlist_count > 0 ? ` · chờ ${s.waitlist_count}` : ''}
      </div>
      {s.status === 'CANCELLED' && <div>Đã hủy: {s.cancel_reason}</div>}
    </div>
  );
}

/**
 * Lịch tuần dạng lưới thời gian: 7 cột ngày, trục giờ theo giờ mở cửa chi nhánh,
 * mỗi buổi là một khối định vị theo giờ bắt đầu + thời lượng, tô màu theo loại lớp.
 */
export default function WeekCalendar({
  weekStart,
  schedules,
  openTime,
  closeTime,
  onSelect,
  onCreateAt,
}: WeekCalendarProps) {
  const now = useNow();
  const days = useMemo(
    () => Array.from({ length: 7 }, (_, i) => weekStart.add(i, 'day')),
    [weekStart],
  );

  const byDay = useMemo(() => {
    const map = new Map<string, ClassSchedule[]>();
    schedules.forEach((s) => {
      const key = dayjs(s.start_at).format('YYYY-MM-DD');
      map.set(key, [...(map.get(key) ?? []), s]);
    });
    return map;
  }, [schedules]);

  const [rangeStart, rangeEnd] = useMemo(() => {
    let start = toMin(openTime, 6 * 60);
    let end = toMin(closeTime, 22 * 60);
    schedules.forEach((s) => {
      const st = minuteOfDay(dayjs(s.start_at));
      start = Math.min(start, st);
      end = Math.max(end, st + s.duration_min);
    });
    return [Math.floor(start / 60) * 60, Math.min(Math.ceil(end / 60) * 60, 24 * 60)];
  }, [openTime, closeTime, schedules]);

  const height = (rangeEnd - rangeStart) * MIN_PX;
  const hours = useMemo(() => {
    const list: number[] = [];
    for (let m = rangeStart; m <= rangeEnd; m += 60) list.push(m);
    return list;
  }, [rangeStart, rangeEnd]);

  const handleColumnClick = (day: Dayjs) => (e: MouseEvent<HTMLDivElement>) => {
    if (!onCreateAt || e.target !== e.currentTarget) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const minutes = rangeStart + Math.floor((e.clientY - rect.top) / MIN_PX / 30) * 30;
    onCreateAt(day.startOf('day').add(minutes, 'minute'));
  };

  // Cuộn tới buổi sớm nhất trong tuần (hoặc giờ hiện tại) khi đổi tuần.
  const scrollRef = useRef<HTMLDivElement>(null);
  const firstMin = useMemo(() => {
    const mins = schedules.map((s) => minuteOfDay(dayjs(s.start_at)));
    return mins.length ? Math.min(...mins) : minuteOfDay(dayjs());
  }, [schedules]);
  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = Math.max(0, (firstMin - rangeStart - 30) * MIN_PX);
  }, [weekStart, firstMin, rangeStart]);

  const gridStyle = { '--wc-hour': `${HOUR_PX}px` } as CSSProperties;

  return (
    <div className="wc-scroll" ref={scrollRef}>
      <div className="wc" style={gridStyle}>
        <div className="wc-head" />
        {days.map((d) => {
          const count = byDay.get(d.format('YYYY-MM-DD'))?.length ?? 0;
          return (
            <div key={d.valueOf()} className={`wc-head${d.isSame(now, 'day') ? ' is-today' : ''}`}>
              <div className="wc-dow">{weekdayName(d)}</div>
              <div className="wc-date">{d.format('DD')}</div>
              <div className="wc-count">{count ? `${count} buổi` : ' '}</div>
            </div>
          );
        })}

        <div className="wc-gutter" style={{ height }}>
          {hours.map((m) =>
            m === rangeStart ? null : (
              <span key={m} className="wc-hour-label" style={{ top: (m - rangeStart) * MIN_PX }}>
                {String(Math.floor(m / 60)).padStart(2, '0')}:00
              </span>
            ),
          )}
        </div>

        {days.map((d) => {
          const key = d.format('YYYY-MM-DD');
          const isToday = d.isSame(now, 'day');
          const nowMin = minuteOfDay(now);
          const placed = layoutDay(byDay.get(key) ?? []);
          const canAdd = !!onCreateAt && !d.isBefore(now, 'day');
          return (
            <div
              key={key}
              className={`wc-col${isToday ? ' is-today' : ''}${canAdd ? ' can-add' : ''}`}
              style={{ height }}
              onClick={canAdd ? handleColumnClick(d) : undefined}
              title={canAdd ? 'Bấm vào ô trống để thêm lịch' : undefined}
            >
              {isToday && nowMin >= rangeStart && nowMin <= rangeEnd && (
                <div className="wc-now" style={{ top: (nowMin - rangeStart) * MIN_PX }} />
              )}
              {placed.map(({ s, start, end, lane, lanes }) => {
                const color = s.class_type_color || '#F97316';
                const cancelled = s.status === 'CANCELLED';
                const full = !cancelled && s.booked_count >= s.capacity;
                const past = dayjs(s.end_at).isBefore(now);
                const blockHeight = Math.max((end - start) * MIN_PX - 2, 22);
                const compact = blockHeight < 56;
                const cls = [
                  'wc-event',
                  cancelled && 'is-cancelled',
                  full && 'is-full',
                  past && 'is-past',
                ]
                  .filter(Boolean)
                  .join(' ');
                return (
                  <Tooltip key={s.id} title={<ScheduleSummary s={s} />} mouseEnterDelay={0.4}>
                    <div
                      role="button"
                      tabIndex={0}
                      className={cls}
                      style={{
                        top: (start - rangeStart) * MIN_PX + 1,
                        height: blockHeight,
                        left: `calc(${(lane / lanes) * 100}% + 2px)`,
                        width: `calc(${100 / lanes}% - 4px)`,
                        background: withAlpha(color, 0.9),
                        borderLeftColor: color,
                        color: readableTextColor(color),
                      }}
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelect(s);
                      }}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          onSelect(s);
                        }
                      }}
                    >
                      <div className="wc-title">
                        {compact && `${dayjs(s.start_at).format('HH:mm')} `}
                        {s.class_type_name}
                      </div>
                      {!compact && (
                        <div className="wc-line">
                          {dayjs(s.start_at).format('HH:mm')}–{dayjs(s.end_at).format('HH:mm')}
                          {blockHeight < 78 && ` · ${s.trainer_name}`}
                        </div>
                      )}
                      {blockHeight >= 78 && <div className="wc-line">{s.trainer_name}</div>}
                      {blockHeight >= 94 && <div className="wc-line">{s.room_name}</div>}
                      {blockHeight >= 40 && (
                        <div className="wc-line">
                          {cancelled ? (
                            'Đã hủy'
                          ) : (
                            <>
                              <span className={`wc-badge${full ? ' is-full' : ''}`}>
                                {s.booked_count}/{s.capacity}
                              </span>
                              {s.waitlist_count > 0 && ` +${s.waitlist_count} chờ`}
                            </>
                          )}
                          {blockHeight < 94 && ` · ${s.room_name}`}
                        </div>
                      )}
                    </div>
                  </Tooltip>
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
}
