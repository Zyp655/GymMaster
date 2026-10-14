'use client';

import { useEffect, useMemo, useRef } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ArrowUpTrayIcon, TrashIcon } from '@heroicons/react/24/outline';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDate, initials } from '@/lib/format';
import { looseT } from '@/lib/utils';
import { useUpdateProfileMutation, useUploadAvatarMutation } from '../queries';
import { createProfileSchema, type ProfileFormValues } from '../schema';
import type { Profile } from '../types';

/** Hồ sơ hội viên: chỉ sửa email, địa chỉ, liên hệ khẩn cấp, ảnh đại diện. */
export function ProfileFormCard({ profile }: { profile: Profile }) {
  const t = useTranslations('portal.overview');
  const ts = useTranslations('portal.status.gender');
  const tc = useTranslations('common');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const updateMutation = useUpdateProfileMutation();
  const uploadMutation = useUploadAvatarMutation();

  const schema = useMemo(() => createProfileSchema(looseT(tv)), [tv]);
  const { member } = profile;

  const form = useForm<ProfileFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      email: member.email ?? '',
      address: member.address ?? '',
      emergency_contact_name: member.emergency_contact_name ?? '',
      emergency_contact_phone: member.emergency_contact_phone ?? '',
      avatar_url: member.avatar_url ?? '',
    },
  });

  useEffect(() => {
    form.reset({
      email: member.email ?? '',
      address: member.address ?? '',
      emergency_contact_name: member.emergency_contact_name ?? '',
      emergency_contact_phone: member.emergency_contact_phone ?? '',
      avatar_url: member.avatar_url ?? '',
    });
  }, [member, form]);

  const avatarUrl = useWatch({ control: form.control, name: 'avatar_url' });

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    try {
      const res = await uploadMutation.mutateAsync(file);
      form.setValue('avatar_url', res.url, { shouldDirty: true });
      toast.info(t('avatarUploaded'));
    } catch (error) {
      toastError(error);
    }
  };

  const onSubmit = form.handleSubmit(async values => {
    try {
      await updateMutation.mutateAsync(values);
      toast.success(t('updateSuccess'));
    } catch (error) {
      toastError(error);
    }
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('profileTitle')}</CardTitle>
        <CardDescription>{t('profileDesc')}</CardDescription>
      </CardHeader>

      <div className="flex flex-wrap items-center gap-4 rounded-xl bg-canvas p-4">
        <Avatar className="size-16">
          {avatarUrl ? <AvatarImage src={avatarUrl} alt={member.full_name} /> : null}
          <AvatarFallback>{initials(member.full_name)}</AvatarFallback>
        </Avatar>
        <div className="grid gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              className="hidden"
              onChange={e => void handleFileChange(e)}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={uploadMutation.isPending}
              onClick={() => fileInputRef.current?.click()}
            >
              <ArrowUpTrayIcon />
              {uploadMutation.isPending ? t('uploadingAvatar') : t('uploadAvatar')}
            </Button>
            {avatarUrl ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => form.setValue('avatar_url', '', { shouldDirty: true })}
              >
                <TrashIcon />
                {t('removeAvatar')}
              </Button>
            ) : null}
          </div>
          <p className="text-caption text-tertiary">{t('avatarHint')}</p>
        </div>
      </div>

      <div className="grid gap-4 rounded-xl bg-canvas p-4 sm:grid-cols-2 lg:grid-cols-5">
        <div>
          <p className="text-caption text-secondary">{t('fullName')}</p>
          <p className="text-body font-medium text-primary">{member.full_name}</p>
        </div>
        <div>
          <p className="text-caption text-secondary">{t('phone')}</p>
          <p className="text-body font-medium text-primary">{member.phone || t('notUpdated')}</p>
        </div>
        <div>
          <p className="text-caption text-secondary">{t('gender')}</p>
          <p className="text-body font-medium text-primary">{ts(member.gender)}</p>
        </div>
        <div>
          <p className="text-caption text-secondary">{t('dob')}</p>
          <p className="text-body font-medium text-primary">
            {member.dob ? formatDate(member.dob) : t('notUpdated')}
          </p>
        </div>
        <div>
          <p className="text-caption text-secondary">{t('idCard')}</p>
          <p className="text-body font-medium text-primary">{member.id_card || t('notUpdated')}</p>
        </div>
      </div>

      <Form {...form}>
        <form onSubmit={onSubmit} className="grid gap-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('email')}</FormLabel>
                  <FormControl>
                    <Input {...field} type="email" placeholder={t('emailPlaceholder')} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="address"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('address')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t('addressPlaceholder')} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="emergency_contact_name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('emergencyName')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t('emergencyNamePlaceholder')} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="emergency_contact_phone"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('emergencyPhone')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t('emergencyPhonePlaceholder')} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className="flex justify-end">
            <Button
              type="submit"
              disabled={updateMutation.isPending || uploadMutation.isPending}
            >
              {updateMutation.isPending ? tc('actions.saving') : tc('actions.saveChanges')}
            </Button>
          </div>
        </form>
      </Form>
    </Card>
  );
}
