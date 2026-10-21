'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import dayjs from 'dayjs';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { QRCodeSVG } from 'qrcode.react';
import { toast } from 'sonner';
import {
  CheckCircleIcon,
  ClipboardDocumentCheckIcon,
  ClipboardDocumentIcon,
  DocumentCheckIcon,
  DocumentTextIcon,
  PencilSquareIcon,
  TicketIcon,
  TrashIcon,
  UserPlusIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { RichHtml } from '@/components/ui/rich-html';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDate, formatDateTime } from '@/lib/format';
import { looseT } from '@/lib/utils';
import {
  useBranchesQuery,
  useCreateMemberGuestPassMutation,
  useMemberContractsQuery,
  useMemberGuestPassesQuery,
  useProfileQuery,
  useSignMemberContractMutation,
} from '../queries';
import { createGuestPassSchema, type GuestPassFormValues } from '../schema';
import type { MemberContractItem } from '../types';
import { PortalPageHeader } from './PortalPageHeader';
import { ContractStatusBadge } from './PortalStatusBadge';

export function ContractsView() {
  const t = useTranslations('portal.contracts');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();

  const contractsQuery = useMemberContractsQuery();
  const guestPassesQuery = useMemberGuestPassesQuery();
  const branchesQuery = useBranchesQuery();
  const profileQuery = useProfileQuery();
  const createGuestPassMutation = useCreateMemberGuestPassMutation();

  const [copiedToken, setCopiedToken] = useState<string | null>(null);

  const branches = useMemo(() => branchesQuery.data ?? [], [branchesQuery.data]);
  const defaultBranchId =
    profileQuery.data?.branch?.id ??
    profileQuery.data?.member.home_branch_id ??
    branches[0]?.id ??
    1;

  const schema = useMemo(() => createGuestPassSchema(looseT(tv)), [tv]);
  const form = useForm<GuestPassFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      guest_name: '',
      guest_phone: '',
      branch_id: defaultBranchId,
      valid_date: dayjs().format('YYYY-MM-DD'),
    },
  });

  useEffect(() => {
    if (!form.getValues('branch_id') && defaultBranchId) {
      form.setValue('branch_id', defaultBranchId);
    }
  }, [defaultBranchId, form]);

  if (contractsQuery.isLoading && guestPassesQuery.isLoading) {
    return (
      <div className="grid gap-4">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-56 rounded-2xl" />
        <Skeleton className="h-64 rounded-2xl" />
      </div>
    );
  }

  if (contractsQuery.isError && guestPassesQuery.isError) {
    return (
      <Card>
        <LoadError
          error={contractsQuery.error ?? guestPassesQuery.error}
          onRetry={() => {
            void contractsQuery.refetch();
            void guestPassesQuery.refetch();
          }}
        />
      </Card>
    );
  }

  const contracts = contractsQuery.data ?? [];
  const guestPasses = guestPassesQuery.data ?? [];

  const onSubmitGuestPass = form.handleSubmit(async values => {
    try {
      await createGuestPassMutation.mutateAsync({
        buyer_name: values.guest_name.trim(),
        guest_name: values.guest_name.trim(),
        phone: values.guest_phone.trim(),
        guest_phone: values.guest_phone.trim(),
        branch_id: values.branch_id,
        valid_date: values.valid_date,
      });
      toast.success(t('createGuestPassSuccess'));
      form.reset({
        guest_name: '',
        guest_phone: '',
        branch_id: values.branch_id,
        valid_date: values.valid_date,
      });
    } catch (error) {
      toastError(error);
    }
  });

  async function handleCopyToken(token: string) {
    try {
      await navigator.clipboard.writeText(token);
      setCopiedToken(token);
      toast.success(t('copiedToken'));
      setTimeout(() => setCopiedToken(prev => (prev === token ? null : prev)), 2000);
    } catch {
      toast.success(t('copiedToken'));
    }
  }

  return (
    <div className="grid gap-6">
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />

      {/* Hợp đồng hội viên điện tử */}
      <Card>
        <CardHeader className="flex-row items-center justify-between gap-2">
          <CardTitle>{t('contractsTitle')}</CardTitle>
          <DocumentCheckIcon className="size-5 text-brand-600" />
        </CardHeader>
        {contractsQuery.isLoading ? (
          <div className="grid gap-3">
            <Skeleton className="h-40 rounded-xl" />
          </div>
        ) : contracts.length === 0 ? (
          <EmptyState
            icon={DocumentTextIcon}
            title={t('emptyContractsTitle')}
            description={t('emptyContractsDesc')}
            className="py-6"
          />
        ) : (
          <div className="grid gap-4">
            {contracts.map(contract => (
              <ContractCard key={contract.id} contract={contract} />
            ))}
          </div>
        )}
      </Card>

      {/* Vé khách mời (Guest Pass 1 ngày) */}
      <div className="grid gap-6 lg:grid-cols-12">
        <Card className="lg:col-span-5">
          <CardHeader className="flex-row items-center justify-between gap-2">
            <div>
              <CardTitle>{t('createGuestPassTitle')}</CardTitle>
              <p className="text-caption text-secondary">{t('guestPassesSubtitle')}</p>
            </div>
            <UserPlusIcon className="size-5 shrink-0 text-brand-600" />
          </CardHeader>

          <Form {...form}>
            <form onSubmit={onSubmitGuestPass} className="grid gap-4" noValidate>
              <FormField
                control={form.control}
                name="guest_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('guestNameLabel')}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('guestNamePlaceholder')} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="guest_phone"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('guestPhoneLabel')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type="tel"
                        inputMode="tel"
                        placeholder={t('guestPhonePlaceholder')}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="branch_id"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('branchLabel')}</FormLabel>
                    <Select
                      value={field.value ? String(field.value) : ''}
                      onValueChange={val => field.onChange(Number(val))}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {branches.length > 0 ? (
                          branches.map(b => (
                            <SelectItem key={b.id} value={String(b.id)}>
                              {b.name}
                            </SelectItem>
                          ))
                        ) : (
                          <SelectItem value={String(defaultBranchId)}>
                            {profileQuery.data?.branch?.name || `Chi nhánh #${defaultBranchId}`}
                          </SelectItem>
                        )}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="valid_date"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('validDateLabel')}</FormLabel>
                    <FormControl>
                      <Input {...field} type="date" min={dayjs().format('YYYY-MM-DD')} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Button type="submit" disabled={createGuestPassMutation.isPending}>
                <TicketIcon />
                {createGuestPassMutation.isPending
                  ? t('creatingGuestPass')
                  : t('createGuestPassBtn')}
              </Button>
            </form>
          </Form>
        </Card>

        <Card className="lg:col-span-7">
          <CardHeader className="flex-row items-center justify-between gap-2">
            <CardTitle>{t('guestPassesTitle')}</CardTitle>
            <TicketIcon className="size-5 text-brand-600" />
          </CardHeader>

          {guestPassesQuery.isLoading ? (
            <div className="grid gap-3">
              <Skeleton className="h-32 rounded-xl" />
              <Skeleton className="h-32 rounded-xl" />
            </div>
          ) : guestPasses.length === 0 ? (
            <EmptyState
              icon={TicketIcon}
              title={t('emptyGuestPassesTitle')}
              description={t('emptyGuestPassesDesc')}
              className="py-6"
            />
          ) : (
            <div className="grid gap-4">
              {guestPasses.map(pass => {
                const isUsed = Boolean(pass.used_at);
                const passToken = pass.token || pass.qr_token || '';
                const passName = pass.guest_name || pass.buyer_name || '—';
                const passPhone = pass.guest_phone || pass.phone || '';
                return (
                  <div
                    key={pass.id}
                    className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-line bg-canvas p-4"
                  >
                    <div className="flex min-w-0 items-center gap-4">
                      <div className="shrink-0 rounded-xl bg-white p-2.5 shadow-xs">
                        <QRCodeSVG value={passToken} size={84} />
                      </div>
                      <div className="grid min-w-0 gap-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="min-w-0 text-h4 font-semibold break-words text-primary">
                            {passName}
                          </span>
                          <Badge
                            variant={isUsed ? 'neutral' : 'success'}
                            className="max-w-full shrink whitespace-normal"
                          >
                            {isUsed
                              ? t('usedBadge', { time: formatDateTime(pass.used_at!) })
                              : t('unusedBadge', { date: formatDate(pass.valid_date) })}
                          </Badge>
                        </div>
                        <p className="text-caption text-secondary">
                          {passPhone}
                          {pass.branch_name ? ` · ${pass.branch_name}` : ''}
                        </p>
                        <p className="font-mono text-body font-bold break-all text-brand-600">
                          {passToken}
                        </p>
                      </div>
                    </div>

                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="h-auto max-w-full break-all whitespace-normal"
                      onClick={() => void handleCopyToken(passToken)}
                    >
                      {copiedToken === passToken ? (
                        <ClipboardDocumentCheckIcon />
                      ) : (
                        <ClipboardDocumentIcon />
                      )}
                      {t('copyTokenBtn', { token: passToken })}
                    </Button>
                  </div>
                );
              })}
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}

/** Thẻ hợp đồng hội viên kèm bảng vẽ chữ ký điện tử (Canvas / Typed). */
function ContractCard({ contract }: { contract: MemberContractItem }) {
  const t = useTranslations('portal.contracts');
  const toastError = useToastApiError();
  const signMutation = useSignMemberContractMutation();

  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [isDrawing, setIsDrawing] = useState(false);
  const [hasDrawn, setHasDrawn] = useState(false);
  const [typedSignature, setTypedSignature] = useState('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const contractTitle =
    contract.title || contract.name || contract.template_name || `Hợp đồng #${contract.id}`;
  const contractVersion = contract.version ?? contract.template_version ?? 1;
  const contractStatus = (contract.status === 'SIGNED' ? 'SIGNED' : 'PENDING') as
    'PENDING' | 'SIGNED';
  const contractBody = contract.rendered_html || contract.body_html || contract.content || '';
  const sigImg = contract.signature_data || contract.signature_url;

  function getCoords(
    e: React.MouseEvent<HTMLCanvasElement> | React.TouchEvent<HTMLCanvasElement>,
  ): { x: number; y: number } | null {
    const canvas = canvasRef.current;
    if (!canvas) return null;
    const rect = canvas.getBoundingClientRect();
    // Canvas co giãn theo CSS (w-full) → quy đổi toạ độ màn hình sang toạ độ bản vẽ.
    const sx = rect.width ? canvas.width / rect.width : 1;
    const sy = rect.height ? canvas.height / rect.height : 1;
    if ('touches' in e) {
      const touch = e.touches[0];
      if (!touch) return null;
      return {
        x: (touch.clientX - rect.left) * sx,
        y: (touch.clientY - rect.top) * sy,
      };
    }
    return {
      x: (e.clientX - rect.left) * sx,
      y: (e.clientY - rect.top) * sy,
    };
  }

  function startDraw(e: React.MouseEvent<HTMLCanvasElement> | React.TouchEvent<HTMLCanvasElement>) {
    const pos = getCoords(e);
    const ctx = canvasRef.current?.getContext('2d');
    if (!pos || !ctx) return;
    ctx.strokeStyle = '#111827';
    ctx.lineWidth = 2.2;
    ctx.lineCap = 'round';
    ctx.beginPath();
    ctx.moveTo(pos.x, pos.y);
    setIsDrawing(true);
    setHasDrawn(true);
    setErrorMsg(null);
  }

  function moveDraw(e: React.MouseEvent<HTMLCanvasElement> | React.TouchEvent<HTMLCanvasElement>) {
    if (!isDrawing) return;
    const pos = getCoords(e);
    const ctx = canvasRef.current?.getContext('2d');
    if (!pos || !ctx) return;
    ctx.lineTo(pos.x, pos.y);
    ctx.stroke();
  }

  function stopDraw() {
    setIsDrawing(false);
  }

  function clearCanvas() {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext('2d');
    if (!canvas || !ctx) return;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    setHasDrawn(false);
  }

  function buildSignatureData(): string | null {
    if (hasDrawn && canvasRef.current) {
      return canvasRef.current.toDataURL('image/png');
    }
    const trimmed = typedSignature.trim();
    if (!trimmed) return null;
    if (typeof document !== 'undefined') {
      const offscreen = document.createElement('canvas');
      offscreen.width = 420;
      offscreen.height = 120;
      const ctx = offscreen.getContext('2d');
      if (ctx) {
        ctx.fillStyle = '#ffffff';
        ctx.fillRect(0, 0, offscreen.width, offscreen.height);
        ctx.fillStyle = '#111827';
        ctx.font = 'italic 600 28px Georgia, serif';
        ctx.fillText(trimmed, 24, 72);
        return offscreen.toDataURL('image/png');
      }
    }
    // API chỉ nhận ảnh PNG (data:image/png;base64) – không có canvas thì không tạo được chữ ký.
    return null;
  }

  async function handleSign() {
    const signatureData = buildSignatureData();
    if (!signatureData) {
      setErrorMsg(t('signRequiredError'));
      return;
    }
    setErrorMsg(null);
    try {
      await signMutation.mutateAsync({
        id: contract.id,
        payload: {
          signature_url: signatureData,
          signature_data: signatureData,
          signature: signatureData,
        },
      });
      toast.success(t('signSuccess'));
    } catch (error) {
      toastError(error);
    }
  }

  return (
    <div className="grid gap-4 rounded-xl border border-line bg-canvas p-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h3 className="min-w-0 text-h4 font-semibold break-words text-primary">
            {contractTitle}
          </h3>
          <Badge variant="outline" className="shrink-0">
            {t('versionLabel', { version: contractVersion })}
          </Badge>
        </div>
        <ContractStatusBadge status={contractStatus} />
      </div>

      <RichHtml
        html={contractBody}
        className="max-h-96 overflow-y-auto rounded-xl border border-line bg-surface p-4 text-body text-secondary"
      />

      {contractStatus === 'SIGNED' ? (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-success-50 p-3.5 text-success">
          <div className="flex items-center gap-2 text-body font-medium">
            <CheckCircleIcon className="size-5 shrink-0" />
            <span>
              {contract.signed_at
                ? t('signedAtLabel', { time: formatDateTime(contract.signed_at) })
                : t('signSuccess')}
            </span>
          </div>
          {sigImg?.startsWith('data:image/') ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={sigImg}
              alt={contractTitle}
              className="h-12 rounded-md border border-line bg-white px-2 py-1"
            />
          ) : null}
        </div>
      ) : (
        <div className="grid gap-3 rounded-xl border border-line bg-surface p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-body font-semibold text-primary">{t('signSectionTitle')}</p>
            <Button type="button" variant="ghost" size="sm" onClick={clearCanvas}>
              <TrashIcon />
              {t('clearCanvasBtn')}
            </Button>
          </div>
          <p className="text-caption text-secondary">{t('signCanvasHint')}</p>

          <canvas
            ref={canvasRef}
            width={520}
            height={140}
            onMouseDown={startDraw}
            onMouseMove={moveDraw}
            onMouseUp={stopDraw}
            onMouseLeave={stopDraw}
            onTouchStart={startDraw}
            onTouchMove={moveDraw}
            onTouchEnd={stopDraw}
            className="h-36 w-full max-w-full min-w-0 cursor-crosshair touch-none rounded-xl border border-dashed border-line bg-white"
          />

          <div className="grid gap-1.5">
            <label className="text-caption font-medium text-secondary">
              {t('typedSignatureLabel')}
            </label>
            <Input
              value={typedSignature}
              onChange={e => {
                setTypedSignature(e.target.value);
                setErrorMsg(null);
              }}
              placeholder={t('typedSignaturePlaceholder')}
            />
          </div>

          {errorMsg ? <p className="text-caption text-danger">{errorMsg}</p> : null}

          <div className="flex justify-end">
            <Button
              type="button"
              disabled={signMutation.isPending}
              onClick={() => void handleSign()}
            >
              <PencilSquareIcon />
              {signMutation.isPending ? t('submittingSign') : t('submitSignBtn')}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
