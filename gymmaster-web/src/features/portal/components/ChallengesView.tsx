'use client';

import { useState } from 'react';
import dayjs from 'dayjs';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { FireIcon, SparklesIcon, TrophyIcon } from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDate, formatNumber } from '@/lib/format';
import { looseT } from '@/lib/utils';
import { extractLeaderboardEntries } from '../api';
import {
  useJoinChallengeMutation,
  useMemberChallengesQuery,
  useMemberLeaderboardQuery,
} from '../queries';
import type { ChallengeItem } from '../types';
import { PortalPageHeader } from './PortalPageHeader';

export function ChallengesView() {
  const t = useTranslations('portal.challenges');
  const tl = looseT(t);
  const toastError = useToastApiError();

  const challengesQuery = useMemberChallengesQuery();
  const leaderboardQuery = useMemberLeaderboardQuery();
  const joinMutation = useJoinChallengeMutation();
  const [joiningId, setJoiningId] = useState<number | null>(null);

  if (challengesQuery.isLoading && leaderboardQuery.isLoading) {
    return (
      <div className="grid gap-4">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-52 rounded-2xl" />
        <Skeleton className="h-64 rounded-2xl" />
      </div>
    );
  }

  if (challengesQuery.isError && leaderboardQuery.isError) {
    return (
      <Card>
        <LoadError
          error={challengesQuery.error ?? leaderboardQuery.error}
          onRetry={() => {
            void challengesQuery.refetch();
            void leaderboardQuery.refetch();
          }}
        />
      </Card>
    );
  }

  const challenges = challengesQuery.data ?? [];
  const leaderboard = extractLeaderboardEntries(leaderboardQuery.data);
  const today = dayjs().format('YYYY-MM-DD');

  async function handleJoin(challenge: ChallengeItem) {
    setJoiningId(challenge.id);
    try {
      await joinMutation.mutateAsync(challenge.id);
      toast.success(t('joinSuccess'));
    } catch (error) {
      toastError(error);
    } finally {
      setJoiningId(null);
    }
  }

  function translateMetric(metric: string): string {
    const known = ['CHECKINS', 'CLASSES', 'PT_SESSIONS'];
    const upper = metric.toUpperCase();
    if (known.includes(upper)) {
      return tl(`metrics.${upper}`);
    }
    return metric;
  }

  return (
    <div className="grid gap-6">
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />

      {/* Thử thách phòng tập đang diễn ra */}
      <Card>
        <CardHeader className="flex-row items-center justify-between gap-2">
          <CardTitle>{t('challengesTitle')}</CardTitle>
          <FireIcon className="size-5 text-brand-600" />
        </CardHeader>
        {challengesQuery.isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2">
            <Skeleton className="h-44 rounded-xl" />
            <Skeleton className="h-44 rounded-xl" />
          </div>
        ) : challenges.length === 0 ? (
          <EmptyState
            icon={FireIcon}
            title={t('emptyChallengesTitle')}
            description={t('emptyChallengesDesc')}
            className="py-6"
          />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            {challenges.map(ch => {
              const targetVal = ch.target_value ?? ch.target ?? 1;
              const target = Math.max(targetVal, 1);
              const progress = Math.max(ch.progress ?? 0, 0);
              const percent = Math.min(Math.round((progress / target) * 100), 100);
              const isCompleted = Boolean(ch.completed || ch.completed_at || progress >= target);
              const isJoining = joiningId === ch.id && joinMutation.isPending;
              const metricType = ch.metric_type || ch.metric || 'CHECKINS';
              // Đã qua ngày kết thúc → không cho tham gia (API cũng chặn).
              const isEnded = Boolean(ch.end_date) && ch.end_date.slice(0, 10) < today;

              return (
                <div
                  key={ch.id}
                  className="flex flex-col justify-between gap-4 rounded-xl border border-line bg-canvas p-4"
                >
                  <div className="grid gap-2">
                    <div className="flex flex-wrap items-start justify-between gap-2">
                      <p className="text-h4 font-semibold text-primary">{ch.name}</p>
                      <Badge variant="brand">
                        {t('rewardBadge', { points: formatNumber(ch.reward_points) })}
                      </Badge>
                    </div>
                    {ch.description ? (
                      <p className="text-body text-secondary">{ch.description}</p>
                    ) : null}
                    <div className="flex flex-wrap items-center gap-2 text-caption text-tertiary">
                      <Badge variant="outline">{translateMetric(metricType)}</Badge>
                      <span>
                        {t('period', {
                          start: formatDate(ch.start_date),
                          end: formatDate(ch.end_date),
                        })}
                      </span>
                    </div>
                  </div>

                  <div className="grid gap-2.5">
                    <div className="flex items-center justify-between text-caption text-secondary">
                      <span>
                        {t('progressLabel', {
                          progress,
                          target: targetVal,
                          percent,
                        })}
                      </span>
                      {isCompleted ? (
                        <Badge variant="success">{t('completedBadge')}</Badge>
                      ) : isEnded ? (
                        <Badge variant="neutral">{t('endedBadge')}</Badge>
                      ) : ch.joined ? (
                        <Badge variant="info">{t('joinedBadge')}</Badge>
                      ) : null}
                    </div>
                    <Progress value={percent} />

                    {!ch.joined && !isEnded ? (
                      <Button
                        type="button"
                        size="sm"
                        disabled={isJoining}
                        onClick={() => void handleJoin(ch)}
                      >
                        <SparklesIcon />
                        {isJoining ? t('joining') : t('joinBtn')}
                      </Button>
                    ) : null}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </Card>

      {/* Bảng xếp hạng hội viên */}
      <Card>
        <CardHeader className="flex-row items-center justify-between gap-2">
          <div>
            <CardTitle>{t('leaderboardTitle')}</CardTitle>
            <p className="text-caption text-secondary">{t('leaderboardSubtitle')}</p>
          </div>
          <TrophyIcon className="size-6 text-warning" />
        </CardHeader>
        {leaderboardQuery.isLoading ? (
          <div className="grid gap-2">
            <Skeleton className="h-12 rounded-xl" />
            <Skeleton className="h-12 rounded-xl" />
            <Skeleton className="h-12 rounded-xl" />
          </div>
        ) : leaderboard.length === 0 ? (
          <EmptyState
            icon={TrophyIcon}
            title={t('emptyLeaderboardTitle')}
            description={t('emptyLeaderboardDesc')}
            className="py-6"
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-14 px-2 sm:w-20 sm:px-4">{t('colRank')}</TableHead>
                <TableHead>{t('colMember')}</TableHead>
                <TableHead className="hidden text-right sm:table-cell">
                  {t('colCheckins')}
                </TableHead>
                <TableHead className="px-2 text-right whitespace-nowrap sm:px-4">
                  <span className="hidden sm:inline">{t('colPoints')}</span>
                  <span className="sm:hidden">{t('colPointsShort')}</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {leaderboard.map(entry => {
                const rankVariant =
                  entry.rank === 1
                    ? 'warning'
                    : entry.rank === 2
                      ? 'info'
                      : entry.rank === 3
                        ? 'brand'
                        : 'neutral';
                const checkins = entry.checkins ?? entry.checkin_count;
                const points = entry.points ?? entry.loyalty_points;
                return (
                  <TableRow key={`${entry.rank}-${entry.member_id}`}>
                    <TableCell className="px-2 sm:px-4">
                      <Badge variant={rankVariant}>#{entry.rank}</Badge>
                    </TableCell>
                    <TableCell className="min-w-0 px-2 whitespace-normal sm:px-4">
                      <div className="grid min-w-0 gap-0.5">
                        <span className="font-semibold break-words text-primary">
                          {entry.full_name}
                        </span>
                        {entry.member_code ? (
                          <span className="text-caption text-tertiary">{entry.member_code}</span>
                        ) : null}
                        {/* Mobile: gộp lượt tập vào dưới tên thay cho cột riêng. */}
                        {checkins !== undefined ? (
                          <span className="text-caption text-secondary sm:hidden">
                            {t('checkinsInline', { count: formatNumber(checkins) })}
                          </span>
                        ) : null}
                      </div>
                    </TableCell>
                    <TableCell className="hidden text-right font-medium text-primary sm:table-cell">
                      {checkins !== undefined ? formatNumber(checkins) : '—'}
                    </TableCell>
                    <TableCell className="px-2 text-right font-semibold whitespace-nowrap text-brand-600 sm:px-4">
                      {points !== undefined ? formatNumber(points) : '—'}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
