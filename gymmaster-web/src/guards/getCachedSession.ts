import 'server-only';
import { cache } from 'react';
import { getServerSession } from 'next-auth';
import { authOptions } from './authOptions';

/** Đọc session một lần cho mỗi request (dedupe giữa layout/page/component). */
export const getCachedSession = cache(() => getServerSession(authOptions));
