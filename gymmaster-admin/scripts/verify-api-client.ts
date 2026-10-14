/**
 * Kiểm tra lõi HTTP client (src/utils/apiClient.ts) – không cần thêm dependency.
 * Chạy: `node scripts/verify-api-client.ts [--real]` (Node ≥ 22.18 / 24, tự bỏ type TS).
 *
 * Phần 1 (mock server node:http): refresh single-flight khi nhiều request đồng thời gặp 40101,
 *   xoay vòng refresh token, 40103/40100 → hết phiên, 5xx → onServerError, unwrap envelope.
 * Phần 2 (--real, backend thật ở API_URL, mặc định http://localhost:8000/v1):
 *   login → me → list/tạo/sửa/xóa chi nhánh → refresh → token cũ bị thu hồi → logout.
 */
import http from 'node:http';
import type { AddressInfo } from 'node:net';
import { createApiClient, ApiError } from '../src/utils/apiClient.ts';
import type { TokenStore } from '../src/utils/token.ts';

let passed = 0;
let failed = 0;
function check(name: string, ok: boolean, detail?: unknown) {
  if (ok) {
    passed += 1;
    console.log(`  ✔ ${name}`);
  } else {
    failed += 1;
    console.log(`  ✘ ${name}`, detail ?? '');
  }
}

function memoryTokens(access: string | null, refresh: string | null): TokenStore & { cleared: number } {
  const s = {
    access,
    refresh,
    cleared: 0,
    getAccessToken: () => s.access,
    getRefreshToken: () => s.refresh,
    setTokens: (a: string, r: string) => {
      s.access = a;
      s.refresh = r;
    },
    clear: () => {
      s.access = null;
      s.refresh = null;
      s.cleared += 1;
    },
  };
  return s;
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

// ------------------------------------------------------------------ Phần 1: mock
async function mockSuite() {
  console.log('\n[1] Interceptor với mock server');
  let gen = 1;
  let validAccess = 'access-1';
  let validRefresh = 'refresh-1';
  let refreshHits = 0;

  const server = http.createServer(async (req, res) => {
    const send = (status: number, body: unknown) => {
      res.writeHead(status, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(body));
    };
    let raw = '';
    for await (const chunk of req) raw += chunk;
    const auth = req.headers.authorization ?? '';

    if (req.url === '/v1/auth/refresh') {
      refreshHits += 1;
      await sleep(80);
      const { refresh_token } = JSON.parse(raw || '{}');
      if (refresh_token !== validRefresh) {
        return send(401, { code: 40103, message: 'Phiên đăng nhập không hợp lệ', data: null });
      }
      gen += 1;
      validAccess = `access-${gen}`;
      validRefresh = `refresh-${gen}`;
      return send(200, {
        code: 0,
        message: 'OK',
        data: { access_token: validAccess, refresh_token: validRefresh, expires_in: 900, user: { id: 1 } },
      });
    }
    if (req.url?.startsWith('/v1/protected')) {
      if (auth === `Bearer ${validAccess}`)
        return send(200, { code: 0, message: 'OK', data: { ok: true, branch: req.headers['x-branch-id'] ?? null } });
      if (auth.startsWith('Bearer access-'))
        return send(401, { code: 40101, message: 'Phiên đăng nhập đã hết hạn', data: null });
      return send(401, { code: 40100, message: 'Token không hợp lệ', data: null });
    }
    if (req.url === '/v1/boom') return send(500, { code: 50000, message: 'Lỗi hệ thống', data: null });
    if (req.url === '/v1/conflict') return send(409, { code: 40900, message: 'Mã chi nhánh đã tồn tại', data: null });
    send(404, { code: 40400, message: 'Không tìm thấy đường dẫn', data: null });
  });
  await new Promise<void>((r) => server.listen(0, '127.0.0.1', r));
  const baseURL = `http://127.0.0.1:${(server.address() as AddressInfo).port}/v1`;

  // 1. 5 request đồng thời với access token hết hạn → refresh đúng 1 lần.
  {
    validAccess = 'access-2';
    validRefresh = 'refresh-1';
    gen = 1;
    refreshHits = 0;
    // Access đang giữ = access-1 (hết hạn), refresh-1 còn hiệu lực. (gen=1 → refresh sẽ cấp access-2)
    const tokens = memoryTokens('access-1', 'refresh-1');
    let expired = 0;
    let refreshedEvents = 0;
    const client = createApiClient({
      baseURL,
      tokens,
      getBranchId: () => 7,
      onSessionExpired: () => (expired += 1),
      onTokensRefreshed: () => (refreshedEvents += 1),
    });
    const results = await Promise.all(
      Array.from({ length: 5 }, (_, i) => client.http.get<{ ok: boolean; branch: string }>(`/protected?i=${i}`)),
    );
    check('5 request đồng thời đều thành công sau refresh', results.every((r) => r.ok === true), results);
    check('chỉ gọi /auth/refresh đúng 1 lần (client)', client.stats.refreshCalls === 1, client.stats);
    check('chỉ gọi /auth/refresh đúng 1 lần (server)', refreshHits === 1, refreshHits);
    check('token được xoay vòng', tokens.getAccessToken() === 'access-2' && tokens.getRefreshToken() === 'refresh-2');
    check('onTokensRefreshed gọi 1 lần', refreshedEvents === 1, refreshedEvents);
    check('không hết phiên', expired === 0);
    check('unwrap envelope + gửi X-Branch-Id', results[0].branch === '7', results[0]);

    // 2. Request mang token cũ tới sau khi đã refresh xong → dùng token mới, không refresh lại.
    const before = refreshHits;
    const r = await client.http.get<{ ok: boolean }>('/protected', { headers: { Authorization: 'Bearer access-1' } });
    check('token cũ → thử lại bằng token mới, không refresh thêm', r.ok && refreshHits === before, refreshHits);
  }

  // 3. Refresh token không hợp lệ → hết phiên (40103), xóa token.
  {
    validAccess = 'access-99';
    const tokens = memoryTokens('access-1', 'refresh-bad');
    let expiredWith: ApiError | null = null;
    const client = createApiClient({ baseURL, tokens, onSessionExpired: (e) => (expiredWith = e) });
    const errs = await Promise.allSettled([client.http.get('/protected'), client.http.get('/protected')]);
    const e0 = errs[0].status === 'rejected' ? errs[0].reason : null;
    check('refresh thất bại → ApiError 40103', e0 instanceof ApiError && e0.code === 40103, e0);
    check('onSessionExpired được gọi', expiredWith !== null);
    check('token bị xóa', tokens.getAccessToken() === null && tokens.getRefreshToken() === null);
    check('2 request đồng thời vẫn chỉ refresh 1 lần', client.stats.refreshCalls === 1, client.stats);
  }

  // 4. 40100 (token rác) → hết phiên ngay, không refresh.
  {
    const tokens = memoryTokens('garbage', 'refresh-x');
    let expired = 0;
    const client = createApiClient({ baseURL, tokens, onSessionExpired: () => (expired += 1) });
    const err = await client.http.get('/protected').catch((e: unknown) => e);
    check('40100 → ApiError 40100 + hết phiên, không refresh',
      err instanceof ApiError && err.code === 40100 && expired === 1 && client.stats.refreshCalls === 0, err);
  }

  // 5. 5xx → onServerError; 409 → giữ nguyên message, không hết phiên.
  {
    const tokens = memoryTokens(null, null);
    let serverErrors = 0;
    let expired = 0;
    const client = createApiClient({
      baseURL,
      tokens,
      onServerError: () => (serverErrors += 1),
      onSessionExpired: () => (expired += 1),
    });
    const e500 = await client.http.get('/boom').catch((e: unknown) => e);
    check('500 → ApiError status 500, handled, onServerError',
      e500 instanceof ApiError && e500.status === 500 && e500.handled && serverErrors === 1, e500);
    const silent = await client.http.get('/boom', { silent: true }).catch((e: unknown) => e);
    check('silent: không gọi onServerError', silent instanceof ApiError && serverErrors === 1);
    const e409 = await client.http.post('/conflict', {}).catch((e: unknown) => e);
    check('409 → ApiError 40900 giữ message tiếng Việt',
      e409 instanceof ApiError && e409.code === 40900 && e409.message === 'Mã chi nhánh đã tồn tại' && expired === 0, e409);
    const net = await createApiClient({ baseURL: 'http://127.0.0.1:1/v1', tokens, onServerError: () => (serverErrors += 1) })
      .http.get('/x')
      .catch((e: unknown) => e);
    check('mất kết nối → ApiError code -1 "Không kết nối được máy chủ"',
      net instanceof ApiError && net.code === -1 && net.message === 'Không kết nối được máy chủ', net);
  }

  server.close();
}

// ------------------------------------------------------------------ Phần 2: backend thật
interface LoginData {
  access_token: string;
  refresh_token: string;
  user: { id: number; email: string; roles: string[]; permissions: string[]; branch_ids: number[] };
}
interface Branch {
  id: number;
  code: string;
  name: string;
  open_time: string;
  close_time: string;
  status: string;
}
interface Page<T> {
  items: T[];
  total: number;
  total_page: number;
  current_page: number;
  limit: number;
}

async function realSuite(apiUrl: string) {
  console.log(`\n[2] Luồng thật với backend ${apiUrl}`);
  const tokens = memoryTokens(null, null);
  let expired = 0;
  const client = createApiClient<LoginData['user']>({ baseURL: apiUrl, tokens, onSessionExpired: () => (expired += 1) });
  const api = client.http;

  const bad = await api
    .post('/auth/login', { email: 'admin@gymmaster.vn', password: 'sai-mat-khau' }, { skipAuthRefresh: true })
    .catch((e: unknown) => e);
  check('sai mật khẩu → 40102', bad instanceof ApiError && bad.code === 40102, bad);

  const login = await api.post<LoginData>(
    '/auth/login',
    { email: 'admin@gymmaster.vn', password: 'Admin@123' },
    { skipAuthRefresh: true },
  );
  tokens.setTokens(login.access_token, login.refresh_token);
  check('login OK, roles có OWNER', login.user.roles.includes('OWNER'), login.user.roles);

  const me = await api.get<LoginData['user']>('/auth/me');
  check('me trả đúng user', me.email === 'admin@gymmaster.vn' && Array.isArray(me.permissions), me.email);

  const page = await api.get<Page<Branch>>('/admin/branches', {
    params: { page: 1, limit: 5, order_by: 'name', order_dir: 'ASC' },
  });
  check('list branches có dạng phân trang', Array.isArray(page.items) && typeof page.total === 'number', page);

  const code = `T${Date.now().toString().slice(-6)}`;
  const body = {
    code,
    name: 'Chi nhánh kiểm thử',
    address: '1 Đường Test',
    phone: '0901234567',
    email: 'test@gymmaster.vn',
    open_time: '06:00',
    close_time: '22:00',
    status: 'ACTIVE',
  };
  const badTime = await api
    .post('/admin/branches', { ...body, open_time: '22:00', close_time: '06:00' })
    .catch((e: unknown) => e);
  check('giờ đóng ≤ giờ mở → lỗi validate', badTime instanceof ApiError && badTime.status === 400, badTime);

  const created = await api.post<Branch>('/admin/branches', body);
  check('tạo chi nhánh', created.id > 0 && created.code === code, created);

  const dup = await api.post('/admin/branches', body).catch((e: unknown) => e);
  check('trùng mã → 40900', dup instanceof ApiError && dup.code === 40900, dup);

  const updated = await api.put<Branch>(`/admin/branches/${created.id}`, { ...body, name: 'Chi nhánh kiểm thử (sửa)', status: 'INACTIVE' });
  check('sửa chi nhánh', updated.name.endsWith('(sửa)') && updated.status === 'INACTIVE', updated);

  const filtered = await api.get<Page<Branch>>('/admin/branches', { params: { keyword: code, status: 'INACTIVE' } });
  check('lọc keyword + status', filtered.items.some((b) => b.id === created.id), filtered.total);

  await api.delete(`/admin/branches/${created.id}`);
  const gone = await api.get(`/admin/branches/${created.id}`).catch((e: unknown) => e);
  check('xóa chi nhánh → get trả 40400', gone instanceof ApiError && gone.code === 40400, gone);

  const oldRefresh = tokens.getRefreshToken();
  const oldAccess = tokens.getAccessToken();
  const newAccess = await client.refreshAccessToken();
  check('refresh cấp access token mới', !!newAccess && newAccess === tokens.getAccessToken());
  check('refresh token được xoay vòng', tokens.getRefreshToken() !== oldRefresh);
  void oldAccess;

  const reuse = await createApiClient({ baseURL: apiUrl, tokens: memoryTokens(null, oldRefresh) })
    .refreshAccessToken()
    .catch((e: unknown) => e);
  check('refresh token cũ bị thu hồi → 40103', reuse instanceof ApiError && reuse.code === 40103, reuse);

  const me2 = await api.get<LoginData['user']>('/auth/me');
  check('gọi API bằng access token mới', me2.id === me.id);

  await api.post('/auth/logout', { refresh_token: tokens.getRefreshToken() }, { skipAuthRefresh: true });
  const afterLogout = await client.refreshAccessToken().catch((e: unknown) => e);
  check('sau logout refresh token không dùng được (40103)', afterLogout instanceof ApiError && afterLogout.code === 40103, afterLogout);
  check('không có lần hết phiên ngoài ý muốn', expired === 0, expired);
}

await mockSuite();
if (process.argv.includes('--real')) {
  await realSuite(process.env.API_URL ?? 'http://localhost:8000/v1');
}
console.log(`\nKết quả: ${passed} đạt, ${failed} lỗi`);
process.exit(failed ? 1 : 0);
