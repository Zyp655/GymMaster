import { useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties, MouseEvent, ReactNode } from 'react';
import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { readableTextColor, weekdayName, withAlpha } from '@/routes/classes/constants';
import '@/routes/classes/classes.css';
import './pt.css';

const HOUR_PX = 64;
const MIN_PX = HOUR_PX / 60;

/** Một khối trên lịch (buổi PT hoặc buổi lớp nhóm). */
export interface CalendarEvent {
  key: string;
  start_at: string;
  end_at: string;
  title: string;
  /** Dòng phụ (hiện khi khối đủ cao). */
  lines?: string[];
  color: string;
  /** Khối mờ gạch ngang (đã hủy). */
  cancelled?: boolean;
  /** Khối chỉ đọc, kiểu viền nét đứt (lớp nhóm). */
  readOnly?: boolean;
  tooltip?: ReactNode;
}

export interface PtWeekCalendarProps<T extends CalendarEvent> {
  weekStart: Dayjs;
  events: T[];
  openTime?: string;
  closeTime?: string;
  onSelect?: (e: T) => void;
  /** Bấm ô trống → thời điểm (làm tròn 30 phút). */
  onCreateAt?: (at: Dayjs) => void;
  /** Đơn vị đếm trên tiêu đề cột. */
  countLabel?: string;
}

const toMin = (hhmm: string | undefined, fallback: number) => {
  const m = /^(\d{1,2}):(\d{2})/.exec(hhmm ?? '');
  return m ? Number(m[1]) * 60 + Number(m[2]) : fallback;
};
const minuteOfDay = (d: Dayjs) => d.hour() * 60 + d.minute();
const durationMin = (e: CalendarEvent) =>
  Math.max(dayjs(e.end_at).diff(dayjs(e.start_at), 'minute'), 15);

interface Placed<T> {
  e: T;
  start: number;
  end: number;
  lane: number;
  lanes: number;
}

/** Xếp các khối chồng giờ trong một ngày thành các làn song song. */
function layoutDay<T extends CalendarEvent>(items: T[]): Placed<T>[] {
  const sorted = items
    .map((e) => {
      const start = minuteOfDay(dayjs(e.start_at));
      return { e, start, end: start + durationMin(e), lane: 0, lanes: 1 };
    })
    .sort((a, b) => a.start - b.start || b.end - a.end);
  const result: Placed<T>[] = [];
  let cluster: Placed<T>[] = [];
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

/**
 * Lưới tuần dùng chung cho lịch PT / lịch dạy HLV – cùng bố cục + CSS với lịch lớp
 * (`routes/classes/WeekCalendar.tsx`) nhưng nhận khối tổng quát.
 */
export default function PtWeekCalendar<T extends CalendarEvent>({
  weekStart,
  events,
  openTime,
  closeTime,
  onSelect,
  onCreateAt,
  countLabel = 'buổi',
}: PtWeekCalendarProps<T>) {
  const now = useNow();
  const days = useMemo(
    () => Array.from({ length: 7 }, (_, i) => weekStart.add(i, 'day')),
    [weekStart],
  );

  const byDay = useMemo(() => {
    const map = new Map<string, T[]>();
    events.forEach((e) => {
      const key = dayjs(e.start_at).format('YYYY-MM-DD');
      map.set(key, [...(map.get(key) ?? []), e]);
    });
    return map;
  }, [events]);

  const [rangeStart, rangeEnd] = useMemo(() => {
    let start = toMin(openTime, 6 * 60);
    let end = toMin(closeTime, 22 * 60);
    events.forEach((e) => {
      const st = minuteOfDay(dayjs(e.start_at));
      start = Math.min(start, st);
      end = Math.max(end, st + durationMin(e));
    });
    return [Math.floor(start / 60) * 60, Math.min(Math.ceil(end / 60) * 60, 24 * 60)];
  }, [openTime, closeTime, events]);

  const height = (rangeEnd - rangeStart) * MIN_PX;
  const hours = useMemo(() => {
    const list: number[] = [];
    for (let m = rangeStart; m <= rangeEnd; m += 60) list.push(m);
    return list;
  }, [rangeStart, rangeEnd]);

  const handleColumnClick = (day: Dayjs) => (ev: MouseEvent<HTMLDivElement>) => {
    if (!onCreateAt || ev.target !== ev.currentTarget) return;
    const rect = ev.currentTarget.getBoundingClientRect();
    const minutes = rangeStart + Math.floor((ev.clientY - rect.top) / MIN_PX / 30) * 30;
    onCreateAt(day.startOf('day').add(minutes, 'minute'));
  };

  const scrollRef = useRef<HTMLDivElement>(null);
  const firstMin = useMemo(() => {
    const mins = events.map((e) => minuteOfDay(dayjs(e.start_at)));
    return mins.length ? Math.min(...mins) : minuteOfDay(dayjs());
  }, [events]);
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
              <div className="wc-count">{count ? `${count} ${countLabel}` : ' '}</div>
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
              title={canAdd ? 'Bấm vào ô trống để đặt buổi' : undefined}
            >
              {isToday && nowMin >= rangeStart && nowMin <= rangeEnd && (
                <div className="wc-now" style={{ top: (nowMin - rangeStart) * MIN_PX }} />
              )}
              {placed.map(({ e, start, end, lane, lanes }) => {
                const past = dayjs(e.end_at).isBefore(now);
                const blockHeight = Math.max((end - start) * MIN_PX - 2, 22);
                const compact = blockHeight < 56;
                const clickable = !!onSelect && !e.readOnly;
                const cls = [
                  'wc-event',
                  e.cancelled && 'is-cancelled',
                  past && 'is-past',
                  e.readOnly && 'pt-readonly',
                ]
                  .filter(Boolean)
                  .join(' ');
                const style: CSSProperties = {
                  top: (start - rangeStart) * MIN_PX + 1,
                  height: blockHeight,
                  left: `calc(${(lane / lanes) * 100}% + 2px)`,
                  width: `calc(${100 / lanes}% - 4px)`,
                  borderLeftColor: e.color,
                  cursor: clickable ? 'pointer' : 'default',
                  ...(e.readOnly
                    ? { background: withAlpha(e.color, 0.15), color: '#1f2937' }
                    : { background: withAlpha(e.color, 0.9), color: readableTextColor(e.color) }),
                };
                const lines = e.lines ?? [];
                const maxLines = Math.max(0, Math.floor((blockHeight - 20) / 16));
                return (
                  <Tooltip key={e.key} title={e.tooltip} mouseEnterDelay={0.4}>
                    <div
                      role={clickable ? 'button' : undefined}
                      tabIndex={clickable ? 0 : undefined}
                      className={cls}
                      style={style}
                      onClick={(ev) => {
                        ev.stopPropagation();
                        if (clickable) onSelect?.(e);
                      }}
                      onKeyDown={(ev) => {
                        if (clickable && (ev.key === 'Enter' || ev.key === ' ')) {
                          ev.preventDefault();
                          onSelect?.(e);
                        }
                      }}
                    >
                      <div className="wc-title">
                        {compact && `${dayjs(e.start_at).format('HH:mm')} `}
                        {e.title}
                      </div>
                      {!compact && (
                        <div className="wc-line">
                          {dayjs(e.start_at).format('HH:mm')}–{dayjs(e.end_at).format('HH:mm')}
                        </div>
                      )}
                      {!compact &&
                        lines.slice(0, Math.max(0, maxLines - 1)).map((l, i) => (
                          <div key={i} className="wc-line">
                            {l}
                          </div>
                        ))}
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
