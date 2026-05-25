import {
  createContext,
  FormEvent,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import {
  ArrowRight,
  BarChart3,
  Box,
  CalendarDays,
  ChevronRight,
  Download,
  FileText,
  Filter,
  Home,
  Layers3,
  LogOut,
  Menu,
  Package,
  Plus,
  Search,
  Sparkles,
  Trash2,
  Upload,
  User,
  X,
  Maximize2,
} from 'lucide-react';
import {
  Link,
  NavLink,
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from 'react-router-dom';

type Session = {
  logged_in: boolean;
  has_auth: boolean;
  account_status: 'guest' | 'authenticated' | 'trial' | 'premium' | 'expired';
  free_trial_active: boolean;
  premium_expired: boolean;
  trial_expires_at?: string | null;
  user_id?: string;
  user_email?: string;
  preferred_currency?: string;
  country_code?: string;
  business_type?: string;
};

type ReportSummary = {
  id: string;
  title: string;
  service_level: number;
  sim_runs: number;
  sim_weeks: number;
  sku_count: number;
  warnings: string[];
  created_at: string;
};

type ReportDetail = ReportSummary & {
  results: AnalysisItem[];
};

type AnalysisItem = {
  parameters: {
    sku: string;
    current_inventory: number;
    lead_time_days: number;
    unit_cost: number;
    order_cost: number;
    holding_cost_rate: number;
  };
  policy: { eoq: number; safety_stock: number; reorder_point: number; service_level: number };
  demand: { weekly_mean: number; annual_demand: number; weekly_std_dev: number };
  simulation: { avg_total_annual_cost: number; avg_stockouts: number };
  forecast: { trend_label: string; seasonality_flag: string };
};

type Sku = {
  user_id: string;
  sku_id: string;
  name: string;
  unit_cost: number;
  order_cost: number;
  holding_pct: number;
  lead_time_days: number;
  selling_price: number;
  current_stock: number;
  created_at: string;
};

type SalesEntry = {
  id: string;
  user_id: string;
  sku_id: string;
  date: string;
  quantity: number;
  created_at: string;
};

type InventoryMovement = {
  id: string;
  user_id: string;
  sku_id: string;
  movement_type: string;
  quantity: number;
  balance_after: number;
  note: string;
  movement_date: string;
  created_at: string;
};

type SkuHistoryResponse = {
  sku: Sku;
  sales: SalesEntry[];
  movements: InventoryMovement[];
};

type CatalogueLocationState = {
  sku?: Sku;
};

type LoadingState = {
  active: boolean;
  message: string;
};

type AnalysisNotice = {
  message: string;
  reportId: string | null;
  completedAt: string;
  elapsedMs: number;
};

type ActivityEvent = {
  id: string;
  kind: string;
  title: string;
  description: string;
  entity_type: string;
  entity_id: string;
  created_at: string;
};

type AppNotification = {
  id: string;
  kind: string;
  title: string;
  body: string;
  report_id: string;
  read_at?: string | null;
  created_at: string;
};

type NotificationSettings = {
  user_id: string;
  enabled: boolean;
  frequency: 'daily' | 'weekly';
  scheduled_time: string;
  email_override: string;
  timezone: string;
  last_sent_at?: string | null;
  updated_at: string;
};

type NotificationListResponse = {
  notifications: AppNotification[];
  unread_count: number;
};

type AnalysisOverlayState = {
  phase: 'loading' | 'complete';
  message: string;
  reportId?: string | null;
  completedAt?: string;
  elapsedMs?: number;
};

type SearchResults = {
  skus: Array<{ sku_id: string; name: string }>;
  reports: ReportSummary[];
  records: Array<{ id: string; template_name: string }>;
};

type Template = {
  id: string;
  name: string;
  description: string;
  has_summary: boolean;
  columns: Array<{ header: string; data_type: string; prefill?: string }>;
};

type CatalogueClassification = {
  sku_id: string;
  abc_class: string;
  xyz_class: string;
  annual_value: number;
  cov: number;
};

type BudgetRecommendation = {
  sku_id: string;
  order_quantity: number;
  cost: number;
  stockout_averted: number;
};

type GeneratedRecord = {
  id: string;
  template_name: string;
  records_count: number;
  created_at: string;
};

type AnalysisResponse = {
  skus_analyzed: number;
  warnings: string[];
  elapsed_ms: number;
  results: AnalysisItem[];
  saved_report_id?: string;
  guest_locked_skus?: number;
};

type ToastTone = 'success' | 'error' | 'info';

type ToastMessage = {
  id: number;
  title: string;
  description?: string;
  tone: ToastTone;
};

type SessionContextValue = {
  session: Session;
  loading: boolean;
  refresh: () => Promise<void>;
};

type ToastContextValue = {
  pushToast: (toast: Omit<ToastMessage, 'id'>) => void;
};

const NAV_ITEMS = [
  { path: '/', label: 'Dashboard', icon: Home },
  { path: '/catalogue', label: 'Catalogue', icon: Package },
  { path: '/records', label: 'Smart Records', icon: Sparkles },
  { path: '/reports', label: 'My Reports', icon: FileText },
  { path: '/upload', label: 'New Analysis', icon: Upload },
] as const;

const CURRENCY_OPTIONS = [
  { value: 'NGN', label: 'NGN - Nigerian naira' },
  { value: 'GHS', label: 'GHS - Ghanaian cedi' },
  { value: 'KES', label: 'KES - Kenyan shilling' },
  { value: 'ZAR', label: 'ZAR - South African rand' },
  { value: 'TZS', label: 'TZS - Tanzanian shilling' },
  { value: 'UGX', label: 'UGX - Ugandan shilling' },
  { value: 'XOF', label: 'XOF - West African CFA franc' },
  { value: 'XAF', label: 'XAF - Central African CFA franc' },
  { value: 'USD', label: 'USD - US dollar' },
  { value: 'EUR', label: 'EUR - Euro' },
  { value: 'GBP', label: 'GBP - British pound' },
  { value: 'CAD', label: 'CAD - Canadian dollar' },
  { value: 'AUD', label: 'AUD - Australian dollar' },
  { value: 'JPY', label: 'JPY - Japanese yen' },
  { value: 'INR', label: 'INR - Indian rupee' },
  { value: 'AED', label: 'AED - UAE dirham' },
] as const;

const COUNTRY_OPTIONS = [
  { value: 'NG', label: 'Nigeria' },
  { value: 'GH', label: 'Ghana' },
  { value: 'KE', label: 'Kenya' },
  { value: 'ZA', label: 'South Africa' },
  { value: 'TZ', label: 'Tanzania' },
  { value: 'UG', label: 'Uganda' },
  { value: 'CI', label: "Cote d'Ivoire" },
  { value: 'CM', label: 'Cameroon' },
  { value: 'US', label: 'United States' },
  { value: 'GB', label: 'United Kingdom' },
  { value: 'CA', label: 'Canada' },
  { value: 'AU', label: 'Australia' },
  { value: 'AE', label: 'United Arab Emirates' },
] as const;

const BUSINESS_TYPE_OPTIONS = [
  { value: 'retail', label: 'Retail', hint: 'Storefront or direct-to-consumer sales' },
  { value: 'wholesale', label: 'Wholesale', hint: 'Bulk orders and longer replenishment cycles' },
  { value: 'ecommerce', label: 'Ecommerce', hint: 'Online catalogues and fast-moving stock' },
  { value: 'mixed', label: 'Mixed', hint: 'A blend of retail, wholesale, and online channels' },
] as const;

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '';
const TOKEN_KEY = 'inventory-optimizer-token';
const BILLING_CHECKOUT_URL = import.meta.env.VITE_BILLING_CHECKOUT_URL ?? '';

const SessionContext = createContext<SessionContextValue | null>(null);
const ToastContext = createContext<ToastContextValue | null>(null);

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers || {});
  headers.set('Accept', 'application/json');
  const token = window.localStorage.getItem(TOKEN_KEY);
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    credentials: 'include',
  });

  if (!response.ok) {
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      const payload = await response.json().catch(() => null) as { error?: string } | null;
      throw new Error(payload?.error || response.statusText);
    }

    throw new Error(await response.text());
  }

  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    return response.json() as Promise<T>;
  }

  return response.text() as Promise<T>;
}

async function downloadFile(path: string, fallbackFilename: string) {
  const headers = new Headers();
  const token = window.localStorage.getItem(TOKEN_KEY);
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method: 'GET',
    headers,
    credentials: 'include',
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || response.statusText);
  }

  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const anchor = window.document.createElement('a');
  anchor.href = url;
  anchor.download = fallbackFilename;
  window.document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.URL.revokeObjectURL(url);
}

function formatMoney(value: number) {
  return `€${value.toFixed(2)}`;
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session>({
    logged_in: false,
    has_auth: false,
    account_status: 'guest',
    free_trial_active: false,
    premium_expired: false,
    trial_expires_at: null,
  });
  const [loading, setLoading] = useState(true);

  const loadSession = useCallback(async () => {
    setLoading(true);
    try {
      const value = await apiFetch<Session>('/api/v1/auth/me');
      setSession(value);
    } catch {
      setSession({
        logged_in: false,
        has_auth: false,
        account_status: 'guest',
        free_trial_active: false,
        premium_expired: false,
        trial_expires_at: null,
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSession();
  }, [loadSession]);

  return (
    <SessionContext.Provider value={{ session, loading, refresh: loadSession }}>
      {children}
    </SessionContext.Provider>
  );
}

function useSession() {
  const value = useContext(SessionContext);
  if (!value) {
    throw new Error('useSession must be used within a SessionProvider');
  }
  return value;
}

function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const pushToast = useCallback((toast: Omit<ToastMessage, 'id'>) => {
    const id = window.setTimeout(() => {
      setToasts((current) => current.filter((item) => item.id !== id));
    }, 4200);

    setToasts((current) => [
      ...current,
      { id, ...toast },
    ]);
  }, []);

  const dismissToast = useCallback((id: number) => {
    window.clearTimeout(id);
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ pushToast }}>
      {children}
      <div className="toast-stack" aria-live="polite" aria-atomic="true">
        {toasts.map((toast) => (
          <div key={toast.id} className={`toast toast--${toast.tone}`}>
            <div>
              <div className="toast__title">{toast.title}</div>
              {toast.description && <div className="toast__description">{toast.description}</div>}
            </div>
            <button className="toast__close" type="button" onClick={() => dismissToast(toast.id)} aria-label="Dismiss notification">
              <X size={16} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

function useToast() {
  const value = useContext(ToastContext);
  if (!value) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return value;
}

function useDebouncedValue<T>(value: T, delay = 250) {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(timer);
  }, [delay, value]);

  return debounced;
}

function AnalysisOverlay({ state, onContinue }: { state: AnalysisOverlayState | null; onContinue?: () => void }) {
  if (!state) {
    return null;
  }

  const isComplete = state.phase === 'complete';

  return (
    <div
      role="dialog"
      aria-live="polite"
      aria-busy="true"
      aria-modal="true"
      style={{ position: 'fixed', inset: 0, zIndex: 6000, background: 'linear-gradient(135deg, rgba(255, 248, 240, 0.92), rgba(237, 247, 244, 0.94))', backdropFilter: 'blur(6px)', pointerEvents: 'all', display: 'grid', placeItems: 'center', padding: 24 }}
    >
      <div style={{ width: 'min(92vw, 620px)', borderRadius: 24, background: 'rgba(255,255,255,0.96)', boxShadow: '0 24px 70px rgba(15, 118, 110, 0.18)', border: '1px solid rgba(15, 118, 110, 0.16)', padding: '1.5rem', textAlign: 'center' }}>
        <div style={{ display: 'grid', placeItems: 'center', gap: '0.9rem' }}>
          <div style={{ width: 64, height: 64, borderRadius: '50%', display: 'grid', placeItems: 'center', background: isComplete ? 'rgba(15, 118, 110, 0.10)' : 'rgba(13, 148, 136, 0.12)', color: '#0f766e' }}>
            <Sparkles size={28} />
          </div>
          <div>
            <div className="page__title" style={{ fontSize: '2rem', marginBottom: '0.4rem' }}>{isComplete ? 'Analysis complete' : 'Analysis running'}</div>
            <p className="page__subtitle" style={{ margin: 0 }}>{state.message}</p>
            {isComplete && state.completedAt && typeof state.elapsedMs === 'number' && (
              <p className="muted" style={{ marginTop: '0.75rem' }}>Finished at {new Date(state.completedAt).toLocaleString()} in {state.elapsedMs} ms.</p>
            )}
            {!isComplete && <p className="muted" style={{ marginTop: '0.75rem' }}>Please wait while the inventory state updates. The app is temporarily locked.</p>}
          </div>
          <div className="toolbar__group" style={{ justifyContent: 'center' }}>
            {isComplete ? (
              <>
                <button className="button button--primary" type="button" onClick={onContinue}>
                  Go to Dashboard
                </button>
                {state.reportId && <Link className="button button--ghost" to={`/reports?highlight=${state.reportId}`}>Open report</Link>}
              </>
            ) : (
              <span className="badge badge--teal">Working...</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function App() {
  return (
    <SessionProvider>
      <ToastProvider>
        <div className="app-shell">
          <Header />
          <main>
            <Routes>
              <Route path="/" element={<HomeRoute />} />
              <Route path="/dashboard" element={<RequireAuth><DashboardPage /></RequireAuth>} />
              <Route path="/catalogue" element={<RequireAuth><CataloguePage /></RequireAuth>} />
              <Route path="/catalogue/:id" element={<RequireAuth><SkuDetailPage /></RequireAuth>} />
              <Route path="/records" element={<RequireAuth><RecordsPage /></RequireAuth>} />
              <Route path="/reports" element={<RequireAuth><ReportsPage /></RequireAuth>} />
              <Route path="/reports/:id" element={<RequireAuth><ReportDetailPage /></RequireAuth>} />
              <Route path="/reports/compare" element={<RequireAuth><ComparePage /></RequireAuth>} />
              <Route path="/upload" element={<UploadPage />} />
              <Route path="/results" element={<ResultsPage />} />
              <Route path="/login" element={<AuthPage mode="login" />} />
              <Route path="/signup" element={<AuthPage mode="signup" />} />
              <Route path="/subscribe" element={<SubscriptionPage />} />
              <Route path="/error" element={<ErrorPage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </main>
          <Footer />
        </div>
      </ToastProvider>
    </SessionProvider>
  );
}

function HomeRoute() {
  const { session, loading } = useSession();

  if (loading) {
    return <LoadingPage />;
  }

  if (session.logged_in) {
    return <DashboardPage />;
  }

  return <LandingPage />;
}

function RequireAuth({ children }: { children: ReactNode }) {
  const { session, loading } = useSession();
  const location = useLocation();

  if (loading) {
    return <LoadingPage />;
  }

  if (!session.logged_in) {
    const redirect = encodeURIComponent(`${location.pathname}${location.search}`);
    return <Navigate to={`/login?redirect=${redirect}`} replace />;
  }

  return <>{children}</>;
}

function LoadingPage() {
  return (
    <div className="page">
      <section className="card" style={{ maxWidth: 640, margin: '3rem auto' }}>
        <div className="card__body">
          <div className="empty">
            <Sparkles size={40} />
            <div>Loading the inventory shell...</div>
          </div>
        </div>
      </section>
    </div>
  );
}

function LandingPage() {
  return (
    <div className="page stack">
      <section className="hero">
        <p className="eyebrow">Smart reorder decisions</p>
        <h1 className="hero__title">Stop guessing. Start optimising your inventory.</h1>
        <p className="hero__copy">
          Upload your sales history and product data. In seconds, get clear reorder recommendations backed by Monte-Carlo simulation.
          No spreadsheets, no formulas, no guesswork.
        </p>
        <div className="hero__actions">
          <Link className="button button--primary" to="/upload?mode=guest">
            Get Started
          </Link>
          <Link className="button button--ghost" to="/login">
            Login
          </Link>
          <Link className="button button--secondary" to="/signup">
            Signup
          </Link>
        </div>
        <p className="muted">Free to use · No sign-up required · Your data stays on your machine</p>
      </section>

      <section className="card">
        <div className="card__body">
          <div className="toolbar">
            <div>
              <p className="eyebrow">Why it matters</p>
              <h2 className="card__title">The problem every seller faces</h2>
            </div>
          </div>
          <div className="grid-3">
            <article className="card stat">
              <div className="card__body">
                <div className="stat__label">Stockouts</div>
                <p className="muted">Running out of best-sellers means lost revenue and lower marketplace rankings.</p>
              </div>
            </article>
            <article className="card stat">
              <div className="card__body">
                <div className="stat__label">Overstock</div>
                <p className="muted">Too much inventory ties up cash and racks up storage fees month after month.</p>
              </div>
            </article>
            <article className="card stat">
              <div className="card__body">
                <div className="stat__label">Gut-feel reordering</div>
                <p className="muted">Order-when-it-looks-low ignores demand variability, lead times, and cost trade-offs.</p>
              </div>
            </article>
          </div>
        </div>
      </section>

      <section className="card">
        <div className="card__body">
          <div className="toolbar">
            <div>
              <p className="eyebrow">What you get</p>
              <h2 className="card__title">For each SKU</h2>
            </div>
          </div>
          <div className="grid-3">
            {[
              ['1', 'Reorder Point', 'The exact inventory level at which you should place your next order.'],
              ['2', 'Order Quantity (EOQ)', 'The optimal number of units per order, balancing ordering and holding costs.'],
              ['3', 'Safety Stock', 'Buffer inventory that protects you against demand variability during lead time.'],
              ['4', 'Expected Stockouts', 'How many stockout events to expect per year with the recommended policy.'],
              ['5', 'Annual Cost Estimate', 'Projected holding and ordering costs so you can see the financial impact.'],
              ['6', 'Plain-English Advice', 'A clear recommendation you can act on — no statistics degree required.'],
            ].map(([number, title, text]) => (
              <article key={title} className="card stat">
                <div className="card__body">
                  <div className="badge badge--stone">{number}</div>
                  <h3 className="card__title" style={{ marginTop: '0.75rem' }}>{title}</h3>
                  <p className="muted">{text}</p>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="card">
        <div className="card__body">
          <div className="toolbar">
            <div>
              <p className="eyebrow">How it works</p>
              <h2 className="card__title">Upload, analyse, review</h2>
            </div>
          </div>
          <div className="grid-3">
            {[
              ['Upload two CSV files', 'Weekly sales history and product parameters.'],
              ['Engine crunches the numbers', 'Demand analysis, safety stock, EOQ, and Monte-Carlo simulation.'],
              ['Review and download', 'See report cards and export the analysis when you are signed in.'],
            ].map(([title, text], index) => (
              <article key={title} className="card stat">
                <div className="card__body">
                  <div className="badge badge--teal">Step {index + 1}</div>
                  <h3 className="card__title" style={{ marginTop: '0.75rem' }}>{title}</h3>
                  <p className="muted">{text}</p>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="hero" style={{ background: 'linear-gradient(135deg, rgba(217, 119, 6, 0.95), rgba(15, 118, 110, 0.95))' }}>
        <h2 className="hero__title" style={{ fontSize: '1.8rem' }}>Ready to optimise?</h2>
        <p className="hero__copy">Start in guest mode, or log in when you are ready to save reports, compare analyses, and use Smart Records.</p>
        <div className="hero__actions">
          <Link className="button button--primary" to="/upload?mode=guest">Get Started</Link>
          <Link className="button button--ghost" to="/login">Login</Link>
        </div>
      </section>
    </div>
  );
}

function NotFoundPage() {
  return (
    <StatusPage
      title="Page not found"
      message="The route you requested does not exist."
      actions={(
        <>
          <Link className="button button--primary" to="/">Go home</Link>
          <Link className="button button--ghost" to="/upload">Run analysis</Link>
        </>
      )}
    />
  );
}

function ErrorPage() {
  return (
    <StatusPage
      title="Something went wrong"
      message="The page could not be loaded. Please try again."
      actions={<Link className="button button--primary" to="/">Back home</Link>}
    />
  );
}

function StatusPage({ title, message, actions }: { title: string; message: string; actions?: ReactNode }) {
  return (
    <div className="page">
      <section className="card status-shell">
        <div className="card__body">
          <EmptyState icon={<FileText size={42} />} title={title} message={message} actions={actions ?? <Link className="button button--primary" to="/">Back home</Link>} />
        </div>
      </section>
    </div>
  );
}

function EmptyState({ icon, title, message, hint, actions }: { icon?: ReactNode; title: string; message: string; hint?: string; actions?: ReactNode }) {
  return (
    <div className="empty-state">
      {icon && <div className="empty-state__icon">{icon}</div>}
      <h2 className="card__title">{title}</h2>
      <p className="muted">{message}</p>
      {hint && <p className="empty-state__hint">Next: {hint}</p>}
      {actions && <div className="toolbar__group">{actions}</div>}
    </div>
  );
}

function activityIcon(kind: string) {
  switch (kind) {
    case 'analysis':
      return <BarChart3 size={16} />;
    case 'generated_record':
      return <Layers3 size={16} />;
    case 'sku_edit':
      return <Package size={16} />;
    default:
      return <Sparkles size={16} />;
  }
}

function activityHref(event: ActivityEvent) {
  if (event.entity_type === 'report') {
    return `/reports/${event.entity_id}`;
  }
  if (event.entity_type === 'generated_record') {
    return '/records';
  }
  if (event.entity_type === 'sku') {
    return `/catalogue/${encodeURIComponent(event.entity_id)}`;
  }
  return '';
}

function Header() {
  const [menuOpen, setMenuOpen] = useState(false);
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebouncedValue(query);
  const [results, setResults] = useState<SearchResults | null>(null);
  const [searchError, setSearchError] = useState('');
  const location = useLocation();
  const { session } = useSession();
  const isLanding = location.pathname === '/' && !session.logged_in;
  const isGuestSurface = !session.logged_in && ['/', '/upload', '/results', '/login', '/signup', '/error'].some((path) => location.pathname === path || location.pathname.startsWith('/results'));
  const accessStateLabel = (session.account_status || 'guest').toUpperCase();
  const accessNotice = session.logged_in && (session.free_trial_active || session.premium_expired || session.account_status === 'trial' || session.account_status === 'expired')
    ? {
        tone: session.premium_expired || session.account_status === 'expired' ? 'expired' : 'trial',
        title: session.premium_expired || session.account_status === 'expired' ? 'Trial expired' : 'Free trial active',
        description: session.premium_expired || session.account_status === 'expired'
          ? 'Your 6-month free access has ended. Subscribe to continue using premium features.'
          : `Premium features are available until ${session.trial_expires_at ? formatDate(session.trial_expires_at) : 'your trial ends'}.`,
      }
    : null;

  useEffect(() => {
    if (!session.logged_in || isLanding || isGuestSurface || !debouncedQuery.trim()) {
      setResults(null);
      setSearchError('');
      return;
    }

    let mounted = true;
    apiFetch<SearchResults>(`/api/v1/search?q=${encodeURIComponent(debouncedQuery.trim())}`)
      .then((value) => {
        if (mounted) {
          setResults(value);
          setSearchError('');
        }
      })
      .catch((error: Error) => {
        if (mounted) {
          setResults(null);
          setSearchError(error.message);
        }
      });

    return () => {
      mounted = false;
    };
  }, [debouncedQuery, isGuestSurface, isLanding, session.logged_in]);

  return (
    <header className="topbar">
      <div className="topbar__inner">
        <Link to="/" className="brand">
          <span className="brand__mark">
            <Box size={22} />
          </span>
          <span className="brand__text">
            <span className="brand__name">Inventory Optimizer</span>
            <span className="brand__subtext">Data-driven inventory ops</span>
          </span>
        </Link>

        <div className="topbar__tools">
          {isLanding || isGuestSurface ? (
            <div className="toolbar__group">
              <Link className="button button--ghost button--small" to="/upload?mode=guest">Try Free</Link>
              <Link className="button button--secondary button--small" to="/login">Login</Link>
              <Link className="button button--primary button--small" to="/signup">Signup</Link>
            </div>
          ) : (
            <>
              <div className="search">
                <Search className="search__icon" />
                <input
                  className="search__input"
                  placeholder="Search SKUs, reports, records..."
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  aria-label="Search SKUs, reports, and records"
                />
                {(results || searchError) && (
                  <div className="search__panel" role="listbox" aria-label="Search results">
                    {searchError ? (
                      <div className="search__group">
                        <div className="muted">{searchError}</div>
                      </div>
                    ) : (
                      <>
                        <div className="search__group">
                          <div className="search__label">SKUs</div>
                          {results?.skus?.length ? results.skus.map((item) => (
                            <Link key={item.sku_id} to="/catalogue" className="search__item">
                              <span>
                                <span className="search__title">{item.sku_id}</span>
                                <span className="search__meta">{item.name}</span>
                              </span>
                              <ChevronRight size={16} />
                            </Link>
                          )) : <div className="muted">No matching SKUs</div>}
                        </div>
                        <div className="search__group">
                          <div className="search__label">Reports</div>
                          {results?.reports?.length ? results.reports.map((report) => (
                            <Link key={report.id} to={`/reports?highlight=${report.id}`} className="search__item">
                              <span>
                                <span className="search__title">{report.title}</span>
                                <span className="search__meta">{formatDate(report.created_at)}</span>
                              </span>
                              <ChevronRight size={16} />
                            </Link>
                          )) : <div className="muted">No matching reports</div>}
                        </div>
                        <div className="search__group">
                          <div className="search__label">Records</div>
                          {results?.records?.length ? results.records.map((record) => (
                            <Link key={record.id} to="/records" className="search__item">
                              <span>
                                <span className="search__title">{record.template_name}</span>
                                <span className="search__meta">Generated record</span>
                              </span>
                              <ChevronRight size={16} />
                            </Link>
                          )) : <div className="muted">No matching records</div>}
                        </div>
                      </>
                    )}
                  </div>
                )}
              </div>

              <div className="account">
                <div className="account__identity">
                  <span className="account__email">{session.user_email || 'Guest access'}</span>
                  <Link className="account__status-link" to="/subscribe">
                    <span className={`account__status account__status--${session.account_status}`}>{accessStateLabel}</span>
                  </Link>
                </div>
                <button
                  type="button"
                  className="menu-btn"
                  onClick={() => setMenuOpen((value) => !value)}
                  aria-expanded={menuOpen}
                  aria-label={menuOpen ? 'Close navigation menu' : 'Open navigation menu'}
                >
                  {menuOpen ? <X size={22} /> : <Menu size={22} />}
                </button>
              </div>
            </>
          )}
        </div>
      </div>

      {menuOpen && !(isLanding || isGuestSurface) && (
        <div className="nav-panel">
          <nav className="nav-panel__inner" aria-label="Primary">
            {NAV_ITEMS.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.path}
                  to={item.path}
                  onClick={() => setMenuOpen(false)}
                  className={({ isActive }) =>
                    `nav-panel__link ${isActive || location.pathname === item.path ? 'nav-panel__link--active' : ''}`
                  }
                >
                  <Icon size={18} />
                  <span>{item.label}</span>
                </NavLink>
              );
            })}
            <div className="nav-panel__footer">
              {session.logged_in ? (
                <button
                  type="button"
                  className="nav-panel__button"
                  onClick={() => {
                    window.localStorage.removeItem(TOKEN_KEY);
                    window.location.href = '/';
                  }}
                >
                  <LogOut size={18} />
                  Logout
                </button>
              ) : (
                <div className="toolbar__group">
                  <Link className="button button--secondary button--small" to="/login" onClick={() => setMenuOpen(false)}>
                    Login
                  </Link>
                  <Link className="button button--primary button--small" to="/signup" onClick={() => setMenuOpen(false)}>
                    Signup
                  </Link>
                </div>
              )}
            </div>
          </nav>
        </div>

      )}

      {accessNotice && (
        <div className={`access-banner access-banner--${accessNotice.tone}`}>
          <div className="access-banner__content">
            <strong>{accessNotice.title}</strong>
            <span>{accessNotice.description}</span>
          </div>
          <Link className="button button--small button--secondary" to="/subscribe">
            {session.premium_expired || session.account_status === 'expired' ? 'Subscribe now' : 'Review Access'}
          </Link>
        </div>
      )}
    </header>
  );
}

function DashboardPage() {
  const navigate = useNavigate();
  const { session } = useSession();
  const [reports, setReports] = useState<ReportSummary[]>([]);
  const [skus, setSkus] = useState<Sku[]>([]);
  const [error, setError] = useState<string>('');
  const [analysisNotice, setAnalysisNotice] = useState<AnalysisNotice | null>(null);
  const [latestReportDetail, setLatestReportDetail] = useState<ReportDetail | null>(null);
  const [deletingReportId, setDeletingReportId] = useState('');
  const [notifications, setNotifications] = useState<AppNotification[]>([]);
  const [unreadNotifications, setUnreadNotifications] = useState(0);
  const [activity, setActivity] = useState<ActivityEvent[]>([]);
  const [dashboardReady, setDashboardReady] = useState(false);
  const [walkthroughDismissed, setWalkthroughDismissed] = useState(false);
  const walkthroughKey = session.user_id ? `inventory-optimizer-walkthrough-dismissed:${session.user_id}` : '';

  useEffect(() => {
    if (!walkthroughKey) {
      setWalkthroughDismissed(false);
      return;
    }
    setWalkthroughDismissed(window.localStorage.getItem(walkthroughKey) === '1');
  }, [walkthroughKey]);

  useEffect(() => {
    if (!session.logged_in) {
      setReports([]);
      setSkus([]);
      setLatestReportDetail(null);
      setNotifications([]);
      setUnreadNotifications(0);
      setActivity([]);
      setDashboardReady(false);
      return;
    }

    const storedNotice = window.sessionStorage.getItem('inventory-optimizer-last-analysis-status');
    if (storedNotice) {
      try {
        setAnalysisNotice(JSON.parse(storedNotice) as AnalysisNotice);
      } catch {
        setAnalysisNotice(null);
      }
    } else {
      setAnalysisNotice(null);
    }

    let mounted = true;
    const loadDashboard = async () => {
      try {
        const [reportsData, skusData, notificationData, activityData] = await Promise.all([
          apiFetch<{ reports: ReportSummary[] }>('/api/v1/reports?limit=1000'),
          apiFetch<{ skus: Sku[] }>('/api/v1/catalogue/skus'),
          apiFetch<NotificationListResponse>('/api/v1/notifications?limit=6'),
          apiFetch<{ activity: ActivityEvent[] }>('/api/v1/activity?limit=8'),
        ]);

        if (!mounted) {
          return;
        }

        const nextReports = reportsData.reports ?? [];
        setReports(nextReports);
        setSkus(skusData.skus ?? []);
        setNotifications(notificationData.notifications ?? []);
        setUnreadNotifications(notificationData.unread_count ?? 0);
        setActivity(activityData.activity ?? []);

        if (nextReports[0]) {
          const detail = await apiFetch<ReportDetail>(`/api/v1/reports/${nextReports[0].id}`);
          if (mounted) {
            setLatestReportDetail(detail);
          }
        } else {
          setLatestReportDetail(null);
        }
      } catch (value) {
        if (mounted) {
          setError((value as Error).message);
        }
      } finally {
        if (mounted) {
          setDashboardReady(true);
        }
      }
    };

    loadDashboard();

    return () => {
      mounted = false;
    };
  }, [session.logged_in]);

  const replenishmentAlerts = useMemo(() => {
    const results = latestReportDetail?.results ?? [];
    return results
      .map((result) => {
        const shortage = Math.ceil(Math.max(0, result.policy.reorder_point - result.parameters.current_inventory));
        return {
          sku: result.parameters.sku,
          shortage,
          currentInventory: result.parameters.current_inventory,
          reorderPoint: Math.round(result.policy.reorder_point),
          annualCost: result.simulation.avg_total_annual_cost,
        };
      })
      .filter((item) => item.shortage > 0)
      .sort((a, b) => b.shortage - a.shortage)
      .slice(0, 3);
  }, [latestReportDetail]);

  const deleteReport = async (reportId: string) => {
    if (!window.confirm('Delete this report? This cannot be undone.')) {
      return;
    }

    setDeletingReportId(reportId);
    setError('');
    try {
      await apiFetch(`/api/v1/reports/${reportId}`, { method: 'DELETE' });
      setReports((current) => current.filter((report) => report.id !== reportId));
      if (analysisNotice?.reportId === reportId) {
        setAnalysisNotice(null);
        window.sessionStorage.removeItem('inventory-optimizer-last-analysis-status');
      }
    } catch (value) {
      setError((value as Error).message);
    } finally {
      setDeletingReportId('');
    }
  };

  const markNotificationRead = async (notificationId: string) => {
    try {
      await apiFetch(`/api/v1/notifications/${notificationId}/read`, { method: 'POST' });
      setNotifications((current) => current.map((item) => (item.id === notificationId ? { ...item, read_at: item.read_at ?? new Date().toISOString() } : item)));
      setUnreadNotifications((current) => Math.max(0, current - 1));
    } catch (value) {
      setError((value as Error).message);
    }
  };

  const dismissWalkthrough = () => {
    if (walkthroughKey) {
      window.localStorage.setItem(walkthroughKey, '1');
    }
    setWalkthroughDismissed(true);
  };

  const showWalkthrough = dashboardReady && !error && !walkthroughDismissed && reports.length === 0 && skus.length === 0 && activity.length === 0;

  const firstRunSteps = [
    { title: '1. Upload your files', text: 'Start on Upload and send in sales history plus SKU parameters.' },
    { title: '2. Run analysis', text: 'The engine calculates reorder points, EOQ, and simulated annual cost.' },
    { title: '3. Review the report', text: 'Open My Reports to inspect the saved analysis and compare results.' },
    { title: '4. Unlock premium tools', text: 'Use the catalogue, records, notifications, and scheduling once you sign in.' },
  ];

  return (
    <div className="page stack">
      <section className="hero">
        <p className="eyebrow">Operational control</p>
        <h1 className="hero__title">Welcome back.</h1>
        <p className="hero__copy">
          {session.logged_in
            ? 'Your dashboard is now driven by live backend routes instead of placeholder numbers.'
            : 'Sign in to unlock live reports, catalogue edits, Smart Records, and saved analyses.'}
        </p>
        <div className="hero__actions">
          <button className="button button--primary" type="button" onClick={() => navigate('/upload')}>
            <Sparkles size={18} />
            New Analysis
          </button>
          {!session.logged_in && (
            <Link className="button button--ghost" to="/login">
              Sign in to continue
            </Link>
          )}
        </div>
      </section>

      {analysisNotice && (
        <section className="card" style={{ border: '1px solid rgba(15, 118, 110, 0.2)' }}>
          <div className="card__body stack">
            <div className="toolbar">
              <div>
                <p className="eyebrow">Latest analysis</p>
                <h2 className="card__title">Report ready</h2>
              </div>
              <span className="badge badge--teal">Dashboard</span>
            </div>
            <p className="muted">{analysisNotice.message}</p>
            <p className="muted">Analysis finished at {new Date(analysisNotice.completedAt).toLocaleString()} in {analysisNotice.elapsedMs} ms.</p>
            <div className="toolbar__group">
              <Link className="button button--primary" to={analysisNotice.reportId ? `/reports/${analysisNotice.reportId}` : '/reports'}>
                Go to My Reports
              </Link>
              {analysisNotice.reportId && <Link className="button button--ghost" to={`/reports/${analysisNotice.reportId}`}>Open saved report</Link>}
            </div>
          </div>
        </section>
      )}

      {showWalkthrough && (
        <section className="card" style={{ border: '1px solid rgba(15, 118, 110, 0.18)', background: 'linear-gradient(135deg, rgba(255, 248, 240, 0.92), rgba(237, 247, 244, 0.96))' }}>
          <div className="card__body stack">
            <div className="toolbar">
              <div>
                <p className="eyebrow">First run</p>
                <h2 className="card__title">Get to value in four steps</h2>
              </div>
              <button className="button button--ghost button--small" type="button" onClick={dismissWalkthrough}>
                Dismiss walkthrough
              </button>
            </div>
            <div className="grid-2">
              {firstRunSteps.map((step) => (
                <article key={step.title} className="card" style={{ border: '1px solid rgba(15, 118, 110, 0.10)' }}>
                  <div className="card__body">
                    <div className="badge badge--teal">{step.title}</div>
                    <p className="muted" style={{ marginTop: '0.75rem' }}>{step.text}</p>
                  </div>
                </article>
              ))}
            </div>
            <div className="toolbar__group">
              <Link className="button button--primary" to="/upload">Start with upload</Link>
              <Link className="button button--ghost" to="/subscribe">See premium tools</Link>
            </div>
          </div>
        </section>
      )}

      {latestReportDetail && (
        <section className="card" style={{ border: replenishmentAlerts.length ? '1px solid rgba(220, 38, 38, 0.18)' : '1px solid rgba(15, 118, 110, 0.18)' }}>
          <div className="card__body stack">
            <div className="toolbar">
              <div>
                <p className="eyebrow">Replenishment feed</p>
                <h2 className="card__title">Stock-up alerts from your latest report</h2>
              </div>
              <span className={`badge ${replenishmentAlerts.length ? 'badge--amber' : 'badge--teal'}`}>
                {replenishmentAlerts.length ? `${replenishmentAlerts.length} action item${replenishmentAlerts.length === 1 ? '' : 's'}` : 'All clear'}
              </span>
            </div>

            {replenishmentAlerts.length ? (
              <div className="stack">
                {replenishmentAlerts.map((alert) => (
                  <div key={alert.sku} className="card" style={{ border: '1px solid rgba(220, 38, 38, 0.12)', background: 'linear-gradient(135deg, rgba(254, 242, 242, 0.9), rgba(255, 251, 235, 0.95))' }}>
                    <div className="card__body">
                      <div className="toolbar">
                        <div>
                          <div className="card__title">SKU {alert.sku}</div>
                          <div className="card__meta">
                            {alert.shortage} units below reorder point · stock {alert.currentInventory} vs ROP {alert.reorderPoint}
                          </div>
                        </div>
                        <span className="badge badge--amber">Replenish</span>
                      </div>
                      <div className="toolbar" style={{ marginTop: '0.75rem' }}>
                        <span className="badge badge--stone">Projected annual cost {formatMoney(alert.annualCost)}</span>
                        <Link className="button button--ghost button--small" to={latestReportDetail.id ? `/reports/${latestReportDetail.id}` : '/reports'}>Open report</Link>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="muted">No immediate replenishment alerts. Your latest report does not show any SKUs below reorder point yet.</p>
            )}

            <div className="toolbar__group">
              <Link className="button button--secondary" to={latestReportDetail.id ? `/reports/${latestReportDetail.id}` : '/reports'}>Review latest report</Link>
              <Link className="button button--ghost" to="/catalogue">Check catalogue</Link>
            </div>
          </div>
        </section>
      )}

      {session.logged_in && (
        <section className="card">
          <div className="card__body stack">
            <div className="toolbar">
              <div>
                <p className="eyebrow">Notification center</p>
                <h2 className="card__title">Recent replenishment alerts</h2>
              </div>
              <span className={`badge ${unreadNotifications ? 'badge--amber' : 'badge--teal'}`}>
                {unreadNotifications ? `${unreadNotifications} unread` : 'All read'}
              </span>
            </div>

            {notifications.length ? (
              <div className="stack">
                {notifications.map((notification) => (
                  <div key={notification.id} className="card" style={{ border: notification.read_at ? '1px solid rgba(15, 118, 110, 0.12)' : '1px solid rgba(220, 38, 38, 0.14)', background: notification.read_at ? 'rgba(255,255,255,0.86)' : 'linear-gradient(135deg, rgba(254, 242, 242, 0.92), rgba(255, 251, 235, 0.95))' }}>
                    <div className="card__body">
                      <div className="toolbar">
                        <div>
                          <div className="card__title">{notification.title}</div>
                          <div className="card__meta">{formatDate(notification.created_at)} · {notification.kind}</div>
                        </div>
                        {notification.read_at ? (
                          <span className="badge badge--stone">Read</span>
                        ) : (
                          <button className="button button--ghost button--small" type="button" onClick={() => markNotificationRead(notification.id)}>
                            Mark read
                          </button>
                        )}
                      </div>
                      <p className="muted" style={{ marginTop: '0.65rem' }}>{notification.body}</p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="muted">Your notification center is quiet right now. New replenishment alerts will appear here and in email.</p>
            )}
          </div>
        </section>
      )}

      {session.logged_in && (
        <section className="card">
          <div className="card__body stack">
            <div className="toolbar">
              <div>
                <p className="eyebrow">Recent activity</p>
                <h2 className="card__title">What changed recently</h2>
              </div>
              <span className="badge badge--stone">{activity.length} event{activity.length === 1 ? '' : 's'}</span>
            </div>

            {activity.length ? (
              <div className="stack">
                {activity.map((event) => {
                  const href = activityHref(event);
                  const icon = activityIcon(event.kind);
                  const content = (
                    <div className="card" style={{ border: '1px solid rgba(15, 118, 110, 0.12)' }}>
                      <div className="card__body">
                        <div className="toolbar">
                          <div className="toolbar__group" style={{ gap: '0.65rem' }}>
                            <span className="badge badge--teal">{icon}</span>
                            <div>
                              <div className="card__title">{event.title}</div>
                              <div className="card__meta">{formatDate(event.created_at)} · {event.kind.replaceAll('_', ' ')}</div>
                            </div>
                          </div>
                          {href ? <ArrowRight size={18} /> : <span className="badge badge--stone">Activity</span>}
                        </div>
                        <p className="muted" style={{ marginTop: '0.65rem' }}>{event.description}</p>
                      </div>
                    </div>
                  );

                  return href ? (
                    <Link key={event.id} to={href} style={{ display: 'block' }}>
                      {content}
                    </Link>
                  ) : (
                    <div key={event.id}>{content}</div>
                  );
                })}
              </div>
            ) : (
              <EmptyState
                icon={<Sparkles size={40} />}
                title="No activity yet"
                message="Run an analysis, edit a SKU, or generate a Smart Record to populate this timeline."
                hint="Upload your first CSVs, then come back here to see the change history."
                actions={<button className="button button--primary" type="button" onClick={() => navigate('/upload')}>Run your first analysis</button>}
              />
            )}
          </div>
        </section>
      )}

      {error && <div className="card"><div className="card__body muted">{error}</div></div>}

      {!session.logged_in && (
        <section className="card">
          <div className="card__body muted">
            Guest mode is active. <Link to="/login">Sign in</Link> to unlock live reports, catalogue edits, and saved records.
          </div>
        </section>
      )}

      <section className="grid-3">
        <StatCard label="Saved Reports" value={reports.length.toString()} />
        <StatCard label="SKUs Analysed" value={skus.length.toString()} />
        <StatCard label="Last Analysis" value={reports[0] ? formatDate(reports[0].created_at) : '—'} />
      </section>

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">Recent Reports</h2>
          <p className="card__meta">Pulled from <span className="badge badge--stone">/api/v1/reports</span></p>
        </div>
        <div className="card__body">
          {reports.length ? (
            <div className="stack">
              {reports.slice(0, 4).map((report) => (
                <Link
                  key={report.id}
                  to={`/reports?highlight=${report.id}`}
                  className="card"
                  style={{ display: 'block', border: analysisNotice?.reportId === report.id ? '2px solid #0f766e' : undefined, boxShadow: analysisNotice?.reportId === report.id ? '0 0 0 4px rgba(15, 118, 110, 0.10)' : undefined }}
                >
                  <div className="card__body">
                    <div className="toolbar">
                      <div>
                        <div className="card__title">{report.title}</div>
                        <div className="card__meta">{formatDate(report.created_at)} · {report.sku_count} SKUs · SL {Math.round(report.service_level * 100)}%</div>
                      </div>
                      <ArrowRight size={18} />
                    </div>
                    {analysisNotice?.reportId === report.id && <div className="badge badge--teal" style={{ marginTop: '0.6rem' }}>Just generated</div>}
                    {!!(report.warnings?.length ?? 0) && <div className="badge badge--amber">{report.warnings.length} warning{report.warnings.length > 1 ? 's' : ''}</div>}
                  </div>
                </Link>
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<BarChart3 size={44} />}
              title="You haven’t run any analyses yet"
              message="Upload sales and parameter files to generate your first live analysis."
              hint="Use the Upload button above, then review the saved report from Dashboard."
              actions={<button className="button button--primary" type="button" onClick={() => navigate('/upload')}>Run your first analysis</button>}
            />
          )}
        </div>
      </section>
    </div>
  );
}

function CataloguePage() {
  const { session } = useSession();
  const { pushToast } = useToast();
  const navigate = useNavigate();
  const [skus, setSkus] = useState<Sku[]>([]);
  const [sort, setSort] = useState('sku_asc');
  const [search, setSearch] = useState('');
  const [error, setError] = useState('');
  const [analyticsError, setAnalyticsError] = useState('');
  const [analytics, setAnalytics] = useState<Record<string, CatalogueClassification> | null>(null);
  const [budgetAmount, setBudgetAmount] = useState('500');
  const [budgetRecommendations, setBudgetRecommendations] = useState<BudgetRecommendation[] | null>(null);
  const [budgetStatus, setBudgetStatus] = useState('Enter a budget to calculate');
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState('');
  const [saleModal, setSaleModal] = useState<{ skuId: string; skuName: string; saleDate: string; quantity: string } | null>(null);
  const [replenishModal, setReplenishModal] = useState<{ skuId: string; skuName: string; replenishDate: string; quantity: string; note: string } | null>(null);
  const [productsMaximized, setProductsMaximized] = useState(false);
  const [silentRefreshing, setSilentRefreshing] = useState(false);
  const [analysisOverlay, setAnalysisOverlay] = useState<AnalysisOverlayState | null>(null);
  const actionButtonStyle = { display: 'inline-flex', alignItems: 'center', gap: 6, whiteSpace: 'nowrap' as const };
  const [form, setForm] = useState({
    sku_id: '',
    name: '',
    unit_cost: '',
    order_cost: '',
    holding_pct: '',
    lead_time_days: '',
    selling_price: '',
    current_stock: '',
  });

  const loadSkus = async (options?: { silent?: boolean }) => {
    if (!session.logged_in) {
      setSkus([]);
      return;
    }

    if (options?.silent) {
      setSilentRefreshing(true);
    }

    const params = new URLSearchParams();
    params.set('sort', sort);
    try {
      const value = await apiFetch<{ skus: Sku[] }>(`/api/v1/catalogue/skus?${params.toString()}`);
      setSkus(value.skus);
    } catch (value) {
      setError((value as Error).message);
    } finally {
      setSilentRefreshing(false);
    }
  };

  useEffect(() => {
    void loadSkus();
  }, [sort, session.logged_in]);

  useEffect(() => {
    if (!session.logged_in) {
      setAnalytics(null);
      setBudgetRecommendations(null);
      return;
    }

    apiFetch<Record<string, CatalogueClassification>>('/api/v1/catalogue/abc-xyz')
      .then((value) => {
        setAnalytics(value ?? null);
        setAnalyticsError('');
      })
      .catch((value: Error) => {
        setAnalytics(null);
        setAnalyticsError(value.message);
      });
  }, [session.logged_in]);

  const visibleSkus = skus.filter((sku) => {
    const query = search.trim().toLowerCase();
    if (!query) {
      return true;
    }
    return sku.sku_id.toLowerCase().includes(query) || sku.name.toLowerCase().includes(query);
  });

  const matrixCounts = useMemo(() => {
    const counts: Record<string, number> = { 'A-X': 0, 'A-Y': 0, 'A-Z': 0, 'B-X': 0, 'B-Y': 0, 'B-Z': 0, 'C-X': 0, 'C-Y': 0, 'C-Z': 0 };
    Object.values(analytics || {}).forEach((item) => {
      const key = `${item.abc_class}-${item.xyz_class}`;
      if (key in counts) {
        counts[key] += 1;
      }
    });
    return counts;
  }, [analytics]);

  const budgetTotal = useMemo(() => budgetRecommendations?.reduce((sum, item) => sum + item.cost, 0) ?? 0, [budgetRecommendations]);

  const submitSku = async (event: FormEvent) => {
    event.preventDefault();
    setSaving(true);
    setError('');
    setStatus('');

    try {
      await apiFetch('/api/v1/catalogue/skus', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sku_id: form.sku_id,
          name: form.name,
          unit_cost: Number(form.unit_cost),
          order_cost: Number(form.order_cost),
          holding_pct: Number(form.holding_pct),
          lead_time_days: Number(form.lead_time_days),
          selling_price: Number(form.selling_price),
          current_stock: Number(form.current_stock),
        }),
      });
      setStatus('SKU saved.');
      pushToast({ title: 'SKU saved', description: `${form.sku_id} is now in the catalogue.`, tone: 'success' });
      setForm({ sku_id: '', name: '', unit_cost: '', order_cost: '', holding_pct: '', lead_time_days: '', selling_price: '', current_stock: '' });
      loadSkus();
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Failed to save SKU', description: message, tone: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const deleteSku = async (skuId: string) => {
    setError('');
    try {
      await apiFetch(`/api/v1/catalogue/skus/${encodeURIComponent(skuId)}`, { method: 'DELETE' });
      void loadSkus({ silent: true });
      pushToast({ title: 'SKU deleted', description: skuId, tone: 'info' });
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Delete failed', description: message, tone: 'error' });
    }
  };

  const logSale = async (skuId: string) => {
    const sku = skus.find((item) => item.sku_id === skuId);
    setSaleModal({
      skuId,
      skuName: sku?.name || skuId,
      saleDate: new Date().toISOString().slice(0, 10),
      quantity: '1',
    });
  };

  const replenishSku = async (skuId: string) => {
    const sku = skus.find((item) => item.sku_id === skuId);
    setReplenishModal({
      skuId,
      skuName: sku?.name || skuId,
      replenishDate: new Date().toISOString().slice(0, 10),
      quantity: '1',
      note: '',
    });
  };

  const submitSale = async (event: FormEvent) => {
    event.preventDefault();
    if (!saleModal) {
      return;
    }

    const quantity = Number(saleModal.quantity);
    if (!saleModal.saleDate || !Number.isFinite(quantity) || quantity <= 0) {
      setError('Enter a valid sale date and quantity.');
      return;
    }

    try {
      const response = await apiFetch<{ sku?: Sku }>(`/api/v1/catalogue/skus/${encodeURIComponent(saleModal.skuId)}/sales`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ date: saleModal.saleDate, quantity }),
      });
      setStatus(`Logged sale for ${saleModal.skuId}.`);
      pushToast({ title: 'Sale logged', description: `${saleModal.skuId} · ${quantity} units`, tone: 'success' });
      if (response.sku) {
        setSkus((current) => current.map((item) => (item.sku_id === response.sku?.sku_id ? (response.sku as Sku) : item)));
      } else {
        setSkus((current) => current.map((item) => (item.sku_id === saleModal.skuId ? { ...item, current_stock: Math.max(0, item.current_stock - quantity) } : item)));
      }
      setSaleModal(null);
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Sale log failed', description: message, tone: 'error' });
    }
  };

  const submitReplenishment = async (event: FormEvent) => {
    event.preventDefault();
    if (!replenishModal) {
      return;
    }

    const quantity = Number(replenishModal.quantity);
    if (!replenishModal.replenishDate || !Number.isFinite(quantity) || quantity <= 0) {
      setError('Enter a valid replenishment date and quantity.');
      return;
    }

    try {
      const response = await apiFetch<{ sku?: Sku }>(`/api/v1/catalogue/skus/${encodeURIComponent(replenishModal.skuId)}/replenish`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ date: replenishModal.replenishDate, quantity, note: replenishModal.note }),
      });
      setStatus(`Replenished ${replenishModal.skuId}.`);
      pushToast({ title: 'Stock replenished', description: `${replenishModal.skuId} · +${quantity} units`, tone: 'success' });
      if (response.sku) {
        setSkus((current) => current.map((item) => (item.sku_id === response.sku?.sku_id ? (response.sku as Sku) : item)));
      } else {
        setSkus((current) => current.map((item) => (item.sku_id === replenishModal.skuId ? { ...item, current_stock: item.current_stock + quantity } : item)));
      }
      setReplenishModal(null);
      void loadSkus({ silent: true });
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Replenishment failed', description: message, tone: 'error' });
    }
  };

  const exportCatalogue = async () => {
    try {
      await downloadFile('/api/v1/catalogue/export.csv', 'catalogue.csv');
      pushToast({ title: 'Catalogue exported', description: 'Downloaded CSV inventory snapshot.', tone: 'success' });
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Export failed', description: message, tone: 'error' });
    }
  };

  const optimizeBudget = async (event: FormEvent) => {
    event.preventDefault();
    if (!budgetAmount.trim()) {
      setBudgetStatus('Enter a budget to calculate');
      setBudgetRecommendations(null);
      return;
    }

    setBudgetStatus('Calculating...');
    try {
      const recommendations = await apiFetch<BudgetRecommendation[]>('/api/v1/catalogue/budget-optimize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ budget: Number(budgetAmount) }),
      });
      setBudgetRecommendations(recommendations ?? []);
      setBudgetStatus((recommendations?.length ?? 0) ? '' : 'No SKUs currently need reordering.');
    } catch (value) {
      const message = (value as Error).message;
      setBudgetRecommendations(null);
      setBudgetStatus(message);
    }
  };

  return (
    <div className="page stack">
      <div className="page__header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
        <div>
          <p className="eyebrow">Catalogue</p>
          <h1 className="page__title">Premium SKU Catalogue</h1>
          <p className="page__subtitle">Backed by <span className="badge badge--stone">/api/v1/catalogue/skus</span> and live CRUD actions.</p>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
          <button className="button button--primary" type="button" onClick={async () => {
            setError('');
            setAnalysisOverlay({ phase: 'loading', message: 'Running catalogue analysis and saving the report...' });
            try {
              const response = await apiFetch<AnalysisResponse>('/api/v1/catalogue/analyze', { method: 'POST' });
              if (response.saved_report_id) {
                window.sessionStorage.setItem('inventory-optimizer-last-analysis-status', JSON.stringify({
                  message: `Analysis complete. Saved report ${response.saved_report_id}. Check Dashboard > My Reports to review it.`,
                  reportId: response.saved_report_id,
                  completedAt: new Date().toISOString(),
                  elapsedMs: response.elapsed_ms,
                }));
              }
              void loadSkus({ silent: true });
              setAnalysisOverlay({
                phase: 'complete',
                message: response.saved_report_id
                  ? 'Catalogue analysis is done. Go to Dashboard to find the saved report.'
                  : 'Catalogue analysis is done. Go to Dashboard to review the results.',
                reportId: response.saved_report_id || null,
                completedAt: new Date().toISOString(),
                elapsedMs: response.elapsed_ms,
              });
            } catch (value) {
              const message = (value as Error).message;
              setError(message);
              setAnalysisOverlay(null);
            }
          }}>
            <Sparkles size={18} /> Auto-Analyze Catalogue
          </button>
          <button className="button button--secondary" type="button" onClick={exportCatalogue}>
            <Download size={18} /> Export
          </button>
        </div>
      </div>

      <AnalysisOverlay state={analysisOverlay} onContinue={() => {
        setAnalysisOverlay(null);
        navigate('/dashboard');
      }} />

      {silentRefreshing && (
        <div className="message-panel message-panel--success">
          <div className="message-panel__body">Refreshing catalogue data...</div>
        </div>
      )}

      {(error || status) && <div className={`message-panel ${error ? 'message-panel--error' : 'message-panel--success'}`}><div className="message-panel__body">{error || status}</div></div>}

      {!session.logged_in && (
        <section className="card">
          <div className="card__body muted">
            Catalogue actions require a signed-in session. <Link to="/login">Log in</Link> to load the live SKU route.
          </div>
        </section>
      )}

      <div className="grid-2">
        <section className="card">
          <div className="card__head" style={{ position: 'relative', paddingRight: '3rem' }}>
            <h2 className="card__title">Add New SKU</h2>
            <p className="card__meta">Creates a backend record in the catalogue table.</p>
          </div>
          <div className="card__body">
            <form className="stack" onSubmit={submitSku}>
              <div className="grid-2">
                <Field label="SKU Code"><input className="input" value={form.sku_id} onChange={(event) => setForm({ ...form, sku_id: event.target.value })} required /></Field>
                <Field label="Product Name"><input className="input" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} /></Field>
                <Field label="Unit Cost"><input className="input" type="number" step="0.01" value={form.unit_cost} onChange={(event) => setForm({ ...form, unit_cost: event.target.value })} required /></Field>
                <Field label="Order Cost"><input className="input" type="number" step="0.01" value={form.order_cost} onChange={(event) => setForm({ ...form, order_cost: event.target.value })} required /></Field>
                <Field label="Holding %"><input className="input" type="number" step="0.01" value={form.holding_pct} onChange={(event) => setForm({ ...form, holding_pct: event.target.value })} required /></Field>
                <Field label="Lead Time"><input className="input" type="number" value={form.lead_time_days} onChange={(event) => setForm({ ...form, lead_time_days: event.target.value })} required /></Field>
                <Field label="Selling Price"><input className="input" type="number" step="0.01" value={form.selling_price} onChange={(event) => setForm({ ...form, selling_price: event.target.value })} required /></Field>
                <Field label="Current Stock"><input className="input" type="number" value={form.current_stock} onChange={(event) => setForm({ ...form, current_stock: event.target.value })} required /></Field>
              </div>
              <button className="button button--primary" type="submit" disabled={saving}>
                <Plus size={18} /> {saving ? 'Saving...' : 'Save SKU'}
              </button>
            </form>
          </div>
        </section>

        <section className="card" style={productsMaximized ? { position: 'relative', zIndex: 1, pointerEvents: 'none', opacity: 0.2 } : undefined}>
          <div className="card__head" style={{ position: 'relative', paddingRight: '3.25rem' }}>
            <div className="toolbar">
              <div>
                <h2 className="card__title">Your Products</h2>
                <p className="card__meta">Filtered locally, sourced from the live route.</p>
              </div>
              <div className="toolbar__group">
                <div className="field" style={{ minWidth: 240 }}>
                  <input className="input" placeholder="Search SKUs or filter..." value={search} onChange={(event) => setSearch(event.target.value)} />
                </div>
                <div className="field" style={{ minWidth: 160 }}>
                  <select className="select" value={sort} onChange={(event) => setSort(event.target.value)}>
                    <option value="sku_asc">Sort by SKU</option>
                    <option value="sku_desc">SKU desc</option>
                    <option value="stock_asc">Stock asc</option>
                    <option value="stock_desc">Stock desc</option>
                  </select>
                </div>
                <button
                  className="button button--ghost button--small"
                  type="button"
                  onClick={() => setProductsMaximized(true)}
                  title="Maximize products list"
                  aria-label="Maximize products list"
                  style={{ position: 'absolute', top: 12, right: 12, minWidth: 40, justifyContent: 'center' }}
                >
                  <Maximize2 size={16} />
                </button>
              </div>
            </div>
          </div>
          <div className="card__body">
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>SKU</th>
                    <th>Cost</th>
                    <th>Lead Time</th>
                    <th>Current Stock</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleSkus.map((sku) => (
                    <tr key={sku.sku_id}>
                      <td>
                        <strong>{sku.sku_id}</strong>
                        <div className="muted">{sku.name}</div>
                      </td>
                      <td>{formatMoney(sku.unit_cost)}</td>
                      <td>{sku.lead_time_days} days</td>
                      <td>{sku.current_stock} units</td>
                      <td>
                        <div className="toolbar__group" style={{ display: 'flex', gap: '0.35rem', alignItems: 'center', flexWrap: 'nowrap', whiteSpace: 'nowrap', overflowX: 'auto' }}>
                          <button className="button button--ghost button--small" type="button" onClick={() => logSale(sku.sku_id)} title="Log sale" aria-label="Log sale" style={actionButtonStyle}>
                            <CalendarDays size={16} />
                          </button>
                          <button className="button button--secondary button--small" type="button" onClick={() => replenishSku(sku.sku_id)} title="Replenish" aria-label="Replenish" style={actionButtonStyle}>
                            <Box size={16} />
                          </button>
                          <Link className="button button--ghost button--small" to={`/catalogue/${encodeURIComponent(sku.sku_id)}`} state={{ sku }} title="View history" aria-label="View history" style={actionButtonStyle}>
                            <FileText size={16} />
                          </Link>
                          <button className="button button--danger button--small" type="button" onClick={() => deleteSku(sku.sku_id)} title="Delete" aria-label="Delete" style={actionButtonStyle}>
                            <Trash2 size={16} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {!visibleSkus.length && (
                    <tr>
                      <td colSpan={5}>
                        <div className="empty">No SKUs match this filter. Clear the search or add a SKU to continue.</div>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      </div>

      {saleModal && (
        <div className="modal-backdrop" role="presentation" onClick={() => setSaleModal(null)} style={{ zIndex: 5000 }}>
          <div className="modal-card" role="dialog" aria-modal="true" aria-labelledby="sale-modal-title" onClick={(event) => event.stopPropagation()}>
            <form className="stack" onSubmit={submitSale}>
              <div>
                <p className="eyebrow">Log sale</p>
                <h2 id="sale-modal-title" className="card__title">{saleModal.skuName}</h2>
                <p className="card__meta">Record the sale in-app so it updates the live catalogue history.</p>
              </div>
              <Field label="Sale date (YYYY-MM-DD)">
                <input
                  className="input"
                  type="date"
                  value={saleModal.saleDate}
                  onChange={(event) => setSaleModal((current) => current ? { ...current, saleDate: event.target.value } : current)}
                  required
                />
              </Field>
              <Field label="Quantity sold">
                <input
                  className="input"
                  type="number"
                  min="1"
                  step="1"
                  value={saleModal.quantity}
                  onChange={(event) => setSaleModal((current) => current ? { ...current, quantity: event.target.value } : current)}
                  required
                />
              </Field>
              <div className="toolbar__group">
                <button className="button button--primary" type="submit">Save sale</button>
                <button className="button button--ghost" type="button" onClick={() => setSaleModal(null)}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {replenishModal && (
        <div className="modal-backdrop" role="presentation" onClick={() => setReplenishModal(null)} style={{ zIndex: 5000 }}>
          <div className="modal-card" role="dialog" aria-modal="true" aria-labelledby="replenish-modal-title" onClick={(event) => event.stopPropagation()}>
            <form className="stack" onSubmit={submitReplenishment}>
              <div>
                <p className="eyebrow">Replenish stock</p>
                <h2 id="replenish-modal-title" className="card__title">{replenishModal.skuName}</h2>
                <p className="card__meta">Add units back into the live catalogue and record the movement history.</p>
              </div>
              <Field label="Replenishment date (YYYY-MM-DD)">
                <input
                  className="input"
                  type="date"
                  value={replenishModal.replenishDate}
                  onChange={(event) => setReplenishModal((current) => current ? { ...current, replenishDate: event.target.value } : current)}
                  required
                />
              </Field>
              <Field label="Quantity added">
                <input
                  className="input"
                  type="number"
                  min="1"
                  step="1"
                  value={replenishModal.quantity}
                  onChange={(event) => setReplenishModal((current) => current ? { ...current, quantity: event.target.value } : current)}
                  required
                />
              </Field>
              <Field label="Note">
                <input
                  className="input"
                  value={replenishModal.note}
                  onChange={(event) => setReplenishModal((current) => current ? { ...current, note: event.target.value } : current)}
                  placeholder="Optional restock note"
                />
              </Field>
              <div className="toolbar__group">
                <button className="button button--primary" type="submit">Save replenishment</button>
                <button className="button button--ghost" type="button" onClick={() => setReplenishModal(null)}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {productsMaximized && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(19, 24, 31, 0.42)', zIndex: 3000, padding: 24 }}>
          <div style={{ position: 'relative', height: '100%', background: '#fff', borderRadius: 18, boxShadow: '0 24px 70px rgba(0,0,0,0.28)', overflow: 'auto', padding: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <h2 style={{ margin: 0 }}>Products (expanded)</h2>
            <div style={{ display: 'flex', gap: 8 }}>
              <button className="button button--ghost" type="button" onClick={() => setProductsMaximized(false)}><X size={16} /> Close</button>
            </div>
          </div>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>SKU</th>
                  <th>Cost</th>
                  <th>Lead Time</th>
                  <th>Current Stock</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {visibleSkus.map((sku) => (
                  <tr key={sku.sku_id}>
                    <td>
                      <strong>{sku.sku_id}</strong>
                      <div className="muted">{sku.name}</div>
                    </td>
                    <td>{formatMoney(sku.unit_cost)}</td>
                    <td>{sku.lead_time_days} days</td>
                    <td>{sku.current_stock} units</td>
                    <td>
                      <div style={{ display: 'flex', gap: 8, flexWrap: 'nowrap', whiteSpace: 'nowrap' }}>
                        <button className="button button--ghost button--small" type="button" onClick={() => logSale(sku.sku_id)} title="Log sale" aria-label="Log sale" style={actionButtonStyle}><CalendarDays size={16} /></button>
                        <button className="button button--secondary button--small" type="button" onClick={() => replenishSku(sku.sku_id)} title="Replenish" aria-label="Replenish" style={actionButtonStyle}><Box size={16} /></button>
                        <Link className="button button--ghost button--small" to={`/catalogue/${encodeURIComponent(sku.sku_id)}`} state={{ sku }} title="View history" aria-label="View history" style={actionButtonStyle}><FileText size={16} /></Link>
                        <button className="button button--danger button--small" type="button" onClick={() => deleteSku(sku.sku_id)} title="Delete" aria-label="Delete" style={actionButtonStyle}><Trash2 size={16} /></button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          </div>
        </div>
      )}

      <section className="card">
        <div className="card__body">
          <div className="toolbar">
            <div>
              <p className="eyebrow">Catalogue Analytics</p>
              <h2 className="card__title">ABC / XYZ Matrix</h2>
              <p className="card__meta">Classify value against demand predictability using live sales history.</p>
            </div>
          </div>

          {analyticsError && <div className="muted" style={{ marginBottom: '0.9rem' }}>{analyticsError}</div>}

          <div className="grid-2">
            <article className="card stat">
              <div className="card__body">
                <div className="abc-matrix">
                  <div className="abc-matrix__corner" />
                  <div className="abc-matrix__head">X (Steady)</div>
                  <div className="abc-matrix__head">Y (Variable)</div>
                  <div className="abc-matrix__head">Z (Erratic)</div>
                  <div className="abc-matrix__side">A (High $)</div>
                  <div className="abc-matrix__cell">{matrixCounts['A-X'] ? <strong>{matrixCounts['A-X']}</strong> : '—'}</div>
                  <div className="abc-matrix__cell">{matrixCounts['A-Y'] ? <strong>{matrixCounts['A-Y']}</strong> : '—'}</div>
                  <div className="abc-matrix__cell">{matrixCounts['A-Z'] ? <strong>{matrixCounts['A-Z']}</strong> : '—'}</div>
                  <div className="abc-matrix__side">B (Med $)</div>
                  <div className="abc-matrix__cell">{matrixCounts['B-X'] ? <strong>{matrixCounts['B-X']}</strong> : '—'}</div>
                  <div className="abc-matrix__cell">{matrixCounts['B-Y'] ? <strong>{matrixCounts['B-Y']}</strong> : '—'}</div>
                  <div className="abc-matrix__cell">{matrixCounts['B-Z'] ? <strong>{matrixCounts['B-Z']}</strong> : '—'}</div>
                  <div className="abc-matrix__side">C (Low $)</div>
                  <div className="abc-matrix__cell">{matrixCounts['C-X'] ? <strong>{matrixCounts['C-X']}</strong> : '—'}</div>
                  <div className="abc-matrix__cell">{matrixCounts['C-Y'] ? <strong>{matrixCounts['C-Y']}</strong> : '—'}</div>
                  <div className="abc-matrix__cell">{matrixCounts['C-Z'] ? <strong>{matrixCounts['C-Z']}</strong> : '—'}</div>
                </div>
              </div>
            </article>

            <article className="card stat">
              <div className="card__body stack">
                <div>
                  <div className="stat__label">Smart Restock</div>
                  <h3 className="card__title">Budget Optimizer</h3>
                  <p className="muted">Allocate a fixed budget across SKUs that urgently need reordering.</p>
                </div>
                <form className="stack" onSubmit={optimizeBudget}>
                  <Field label="Budget Amount">
                    <input className="input" type="number" min="1" step="0.01" value={budgetAmount} onChange={(event) => setBudgetAmount(event.target.value)} placeholder="Enter Budget $" />
                  </Field>
                  <button className="button button--primary" type="submit">Optimize</button>
                </form>
                <div className="budget-result">
                  {budgetStatus && <div className="muted">{budgetStatus}</div>}
                  {budgetRecommendations?.length ? (
                    <div className="table-wrap">
                      <table className="table recommendation-table">
                        <thead>
                          <tr>
                            <th>SKU</th>
                            <th>Order Qty</th>
                            <th>Subtotal</th>
                          </tr>
                        </thead>
                        <tbody>
                          {budgetRecommendations.map((item) => (
                            <tr key={item.sku_id}>
                              <td>{item.sku_id}</td>
                              <td>{item.order_quantity}</td>
                              <td>{formatMoney(item.cost)}</td>
                            </tr>
                          ))}
                          <tr>
                            <td colSpan={2}><strong>Total Allocated</strong></td>
                            <td><strong>{formatMoney(budgetTotal)}</strong></td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  ) : null}
                </div>
              </div>
            </article>
          </div>
        </div>
      </section>
    </div>
  );
}

function SkuDetailPage() {
  const { id } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const { pushToast } = useToast();
  const initialDetail = ((location.state as CatalogueLocationState | null)?.sku)
    ? {
        sku: (location.state as CatalogueLocationState).sku as Sku,
        sales: [] as SalesEntry[],
        movements: [] as InventoryMovement[],
      }
    : null;
  const [detail, setDetail] = useState<SkuHistoryResponse | null>(initialDetail);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [replenishForm, setReplenishForm] = useState({ date: new Date().toISOString().slice(0, 10), quantity: '1', note: '' });
  const sku = detail?.sku ?? initialDetail?.sku ?? null;

  useEffect(() => {
    if (!id) {
      setError('Missing SKU identifier.');
      setLoading(false);
      return;
    }

    setLoading(true);
    setError('');
    apiFetch<SkuHistoryResponse>(`/api/v1/catalogue/skus/${encodeURIComponent(id)}`)
      .then((value) => setDetail(value))
      .catch((value: Error) => {
        setError(value.message);
        if (!initialDetail) {
          setDetail(null);
        }
      })
      .finally(() => setLoading(false));
  }, [id]);

  const submitReplenishment = async (event: FormEvent) => {
    event.preventDefault();
    if (!id || !sku) {
      return;
    }

    const quantity = Number(replenishForm.quantity);
    if (!replenishForm.date || !Number.isFinite(quantity) || quantity <= 0) {
      setError('Enter a valid replenishment date and quantity.');
      return;
    }

    setSaving(true);
    setError('');
    try {
      await apiFetch(`/api/v1/catalogue/skus/${encodeURIComponent(id)}/replenish`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ date: replenishForm.date, quantity, note: replenishForm.note }),
      });
      pushToast({ title: 'Stock replenished', description: `${id} · +${quantity} units`, tone: 'success' });
      setReplenishForm((current) => ({ ...current, quantity: '1', note: '' }));
      const refreshed = await apiFetch<SkuHistoryResponse>(`/api/v1/catalogue/skus/${encodeURIComponent(id)}`);
      setDetail(refreshed);
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Replenishment failed', description: message, tone: 'error' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="page stack">
      <div className="page__header">
        <div>
          <p className="eyebrow">Catalogue detail</p>
          <h1 className="page__title">SKU history and live stock</h1>
          <p className="page__subtitle">Sales and replenishment movements are read from the database.</p>
        </div>
        <button className="button button--secondary" type="button" onClick={() => navigate('/catalogue')}>
          <ChevronRight size={18} /> Back to catalogue
        </button>
      </div>

      {(error || loading) && (
        <div className={`message-panel ${error ? 'message-panel--error' : 'message-panel--success'}`}>
          <div className="message-panel__body">{error || 'Loading SKU history...'}</div>
        </div>
      )}

      {!loading && !sku && !error && (
        <section className="card">
          <div className="card__body">
            <EmptyState
              icon={<Box size={44} />}
              title="No SKU history available"
              message="This SKU has no recorded sales or movements, or access is blocked. Try refreshing or check your subscription status."
              actions={<button className="button button--primary" type="button" onClick={() => { setLoading(true); setError(''); apiFetch<SkuHistoryResponse>(`/api/v1/catalogue/skus/${encodeURIComponent(id ?? '')}`).then((v) => setDetail(v)).catch((e: Error) => setError(e.message)).finally(() => setLoading(false)); }}>Refresh</button>}
            />
          </div>
        </section>
      )}

      {sku && (
        <>
          <section className="grid-2">
            <article className="card stat">
              <div className="card__body stack">
                <div>
                  <p className="eyebrow">Current SKU</p>
                  <h2 className="card__title">{sku.sku_id}</h2>
                  <p className="card__meta">{sku.name || 'Unnamed product'}</p>
                </div>
                <div className="feature-grid">
                  <div className="feature-pill">Stock: {sku.current_stock}</div>
                  <div className="feature-pill">Price: {formatMoney(sku.selling_price)}</div>
                  <div className="feature-pill">Lead time: {sku.lead_time_days} days</div>
                  <div className="feature-pill">Cost: {formatMoney(sku.unit_cost)}</div>
                </div>
              </div>
            </article>

            <article className="card">
              <div className="card__body stack">
                <div>
                  <p className="eyebrow">Replenish</p>
                  <h2 className="card__title">Add stock</h2>
                  <p className="card__meta">Updates the current_stock column and writes an inventory movement row.</p>
                </div>
                <form className="stack" onSubmit={submitReplenishment}>
                  <Field label="Date">
                    <input className="input" type="date" value={replenishForm.date} onChange={(event) => setReplenishForm((current) => ({ ...current, date: event.target.value }))} required />
                  </Field>
                  <Field label="Quantity">
                    <input className="input" type="number" min="1" step="1" value={replenishForm.quantity} onChange={(event) => setReplenishForm((current) => ({ ...current, quantity: event.target.value }))} required />
                  </Field>
                  <Field label="Note">
                    <input className="input" value={replenishForm.note} onChange={(event) => setReplenishForm((current) => ({ ...current, note: event.target.value }))} placeholder="Optional note" />
                  </Field>
                  <button className="button button--primary" type="submit" disabled={saving}>{saving ? 'Saving...' : 'Save replenishment'}</button>
                </form>
              </div>
            </article>
          </section>

          <section className="grid-2">
            <article className="card">
              <div className="card__body stack">
                <div>
                  <p className="eyebrow">Sales history</p>
                  <h2 className="card__title">Logged sales</h2>
                </div>
                {(detail?.sales ?? []).length ? (
                  <div className="table-wrap">
                    <table className="table">
                      <thead>
                        <tr>
                          <th>Date</th>
                          <th>Quantity</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(detail?.sales ?? []).map((sale) => (
                          <tr key={sale.id}>
                            <td>{formatDate(sale.date)}</td>
                            <td>{sale.quantity}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <EmptyState
                    icon={<CalendarDays size={44} />}
                    title="No sales have been logged"
                    message="Log sales from the catalogue to populate this history."
                  />
                )}
              </div>
            </article>

            <article className="card">
              <div className="card__body stack">
                <div>
                  <p className="eyebrow">Movement log</p>
                  <h2 className="card__title">Inventory changes</h2>
                </div>
                {(detail?.movements ?? []).length ? (
                  <div className="table-wrap">
                    <table className="table">
                      <thead>
                        <tr>
                          <th>Date</th>
                          <th>Type</th>
                          <th>Qty</th>
                          <th>Balance</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(detail?.movements ?? []).map((movement) => (
                          <tr key={movement.id}>
                            <td>{formatDate(movement.movement_date)}</td>
                            <td>{movement.movement_type}</td>
                            <td>{movement.quantity > 0 ? `+${movement.quantity}` : movement.quantity}</td>
                            <td>{movement.balance_after}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <EmptyState
                    icon={<Box size={44} />}
                    title="No movements yet"
                    message="Sales and replenishments will show up here once they are recorded."
                  />
                )}
              </div>
            </article>
          </section>
        </>
      )}
    </div>
  );
}

function RecordsPage() {
  const { session } = useSession();
  const { pushToast } = useToast();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [history, setHistory] = useState<GeneratedRecord[]>([]);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [templateQuery, setTemplateQuery] = useState('');
  const [templateFilter, setTemplateFilter] = useState('');
  const [selectedTemplateId, setSelectedTemplateId] = useState('');
  const [prefillCatalogue, setPrefillCatalogue] = useState(true);
  const [selectedColumnsByTemplate, setSelectedColumnsByTemplate] = useState<Record<string, string[]>>({});

  const load = () => {
    if (!session.logged_in) {
      setTemplates([]);
      setHistory([]);
      return;
    }

    Promise.all([
      apiFetch<Template[]>('/api/v1/records/templates'),
      apiFetch<GeneratedRecord[]>('/api/v1/records/history'),
    ])
      .then(([templateData, historyData]) => {
        const nextTemplates = templateData ?? [];
        setTemplates(nextTemplates);
        setHistory(historyData ?? []);
        setSelectedTemplateId((current) => current || nextTemplates[0]?.id || '');
      })
      .catch((value: Error) => setError(value.message));
  };

  useEffect(() => {
    load();
  }, [session.logged_in]);

  const visibleTemplates = useMemo(() => {
    const query = templateQuery.trim().toLowerCase();
    return templates.filter((template) => {
      const matchesQuery = !query
        || template.name.toLowerCase().includes(query)
        || template.description.toLowerCase().includes(query)
        || template.id.toLowerCase().includes(query);
      const matchesFilter = !templateFilter || (templateFilter === 'has_summary' ? template.has_summary : true);
      return matchesQuery && matchesFilter;
    });
  }, [templateFilter, templateQuery, templates]);

  const selectedTemplate = useMemo(
    () => templates.find((template) => template.id === selectedTemplateId) || null,
    [selectedTemplateId, templates],
  );

  useEffect(() => {
    if (!selectedTemplate) {
      return;
    }

    const allColumns = (selectedTemplate.columns ?? []).map((column) => column.header);
    setSelectedColumnsByTemplate((current) => {
      if (current[selectedTemplate.id]) {
        return current;
      }

      return {
        ...current,
        [selectedTemplate.id]: allColumns,
      };
    });
  }, [selectedTemplate]);

  const selectedColumns = selectedTemplate ? (selectedColumnsByTemplate[selectedTemplate.id] ?? (selectedTemplate.columns ?? []).map((column) => column.header)) : [];

  const toggleColumn = (columnHeader: string) => {
    if (!selectedTemplate) {
      return;
    }

    setSelectedColumnsByTemplate((current) => {
      const currentSelection = current[selectedTemplate.id] ?? (selectedTemplate.columns ?? []).map((column) => column.header);
      const nextSelection = currentSelection.includes(columnHeader)
        ? currentSelection.filter((value) => value !== columnHeader)
        : [...currentSelection, columnHeader];

      return {
        ...current,
        [selectedTemplate.id]: nextSelection,
      };
    });
  };

  const generate = async (template: Template) => {
    const columns = selectedColumnsByTemplate[template.id] ?? (template.columns ?? []).map((column) => column.header);
    try {
      await apiFetch('/api/v1/records/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          template_id: template.id,
          prefill: prefillCatalogue,
          columns,
        }),
      });
      setStatus(`Generated ${template.name}.`);
      pushToast({ title: 'Record generated', description: template.name, tone: 'success' });
      load();
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Generation failed', description: message, tone: 'error' });
    }
  };

  return (
    <div className="page stack">
      <div className="page__header">
        <div>
          <p className="eyebrow">Smart Records</p>
          <h1 className="page__title">Template generator</h1>
          <p className="page__subtitle">Generate formula-ready Excel sheets from live templates, just like the original records workspace.</p>
        </div>
      </div>

      <section className="card records-banner">
        <div className="card__body">
          <strong>Welcome to Smart Records!</strong> Generate professional, formula-ready Excel sheets instantly. Choose a template below to get started. No Excel expertise required.
        </div>
      </section>

      {(error || status) && <div className="card"><div className="card__body muted">{error || status}</div></div>}

      {!session.logged_in && (
        <section className="card">
          <div className="card__body muted">
            Smart Records is connected to protected routes. <Link to="/login">Sign in</Link> to generate and download live spreadsheets.
          </div>
        </section>
      )}

      <section className="card">
        <div className="card__body stack">
          <div className="toolbar">
            <div>
              <p className="eyebrow">1. Select a Template</p>
              <h2 className="card__title">Pick a sheet layout</h2>
            </div>
            <div className="toolbar__group">
              <Field label="Search templates" style={{ minWidth: 220 }}>
                <input className="input" type="search" placeholder="Search templates..." value={templateQuery} onChange={(event) => setTemplateQuery(event.target.value)} />
              </Field>
              <Field label="Filter" style={{ minWidth: 160 }}>
                <select className="select" value={templateFilter} onChange={(event) => setTemplateFilter(event.target.value)}>
                  <option value="">All</option>
                  <option value="has_summary">Has Summary</option>
                </select>
              </Field>
            </div>
          </div>

          <div className="template-grid">
            {visibleTemplates.map((template) => (
              <button
                key={template.id}
                type="button"
                className={`template-card ${selectedTemplateId === template.id ? 'template-card--selected' : ''}`}
                onClick={() => setSelectedTemplateId(template.id)}
              >
                <div className="template-card__icon">{template.has_summary ? '▣' : '▤'}</div>
                <h3>{template.name}</h3>
                <p>{template.description}</p>
              </button>
            ))}
            {!visibleTemplates.length && (
              <EmptyState
                title="No templates available"
                message="Try clearing the filter or searching a different template name."
              />
            )}
          </div>
        </div>
      </section>

      {selectedTemplate && (
        <section className="card records-builder">
          <div className="card__body stack">
            <div className="toolbar">
              <div>
                <p className="eyebrow">2. Customization Options</p>
                <h2 className="card__title">{selectedTemplate.name}</h2>
                <p className="card__meta">{selectedTemplate.description}</p>
              </div>
              {selectedTemplate.has_summary ? <span className="badge badge--teal">Summary</span> : <span className="badge badge--stone">Sheet</span>}
            </div>

            <div className="grid-3">
              <article className="option-card">
                <div className="stat__label">Date Generated</div>
                <p className="muted">Add timestamp to reports</p>
              </article>
              <article className="option-card">
                <div className="stat__label">Template Formatted</div>
                <p className="muted">Pre-generated records Excel</p>
              </article>
              <article className="option-card">
                <div className="stat__label">Merch Cost</div>
                <p className="muted">Download</p>
              </article>
            </div>

            <label className="field" style={{ width: 'fit-content', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <span className="field__label">Pre-fill with my SKU catalogue data</span>
              <input
                type="checkbox"
                checked={prefillCatalogue}
                onChange={(event) => setPrefillCatalogue(event.target.checked)}
              />
            </label>

            <div>
              <div className="field__label" style={{ marginBottom: '0.65rem' }}>Select Columns to Include</div>
              <div className="columns-list">
                {(selectedTemplate.columns ?? []).map((column) => (
                  <label key={column.header} className="column-item">
                    <input
                      type="checkbox"
                      checked={selectedColumns.includes(column.header)}
                      disabled={column.data_type === 'formula'}
                      onChange={() => toggleColumn(column.header)}
                    />
                    <span>{column.header}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="toolbar__group">
              <button className="button button--primary" type="button" onClick={() => generate(selectedTemplate)}>
                Generate Excel Sheet
              </button>
              <button className="button button--ghost" type="button" onClick={() => setSelectedTemplateId('')}>
                Cancel
              </button>
            </div>
          </div>
        </section>
      )}

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">3. Generation History</h2>
          <p className="card__meta">Downloaded sheets are listed here after generation.</p>
        </div>
        <div className="card__body">
          {history.length ? (
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Date Generated</th>
                    <th>Template Formatted</th>
                    <th>Records Count</th>
                    <th>Download</th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((record) => (
                    <tr key={record.id}>
                      <td>{formatDate(record.created_at)}</td>
                      <td>{record.template_name}</td>
                      <td>{record.records_count}</td>
                      <td>
                        <button
                          className="button button--ghost button--small"
                          type="button"
                          onClick={() => downloadFile(`/records/download/${record.id}`, `${record.template_name.replace(/\s+/g, '_')}.xlsx`)}
                        >
                          Download
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState
              icon={<Layers3 size={40} />}
              title="No sheets generated yet"
              message="Select a template and generate your first workbook to see download history here."
              hint="Choose a template above, then click Generate Excel Sheet to create the first record."
              actions={<button className="button button--primary" type="button" onClick={() => selectedTemplate && generate(selectedTemplate)} disabled={!selectedTemplate}>Generate first sheet</button>}
            />
          )}
        </div>
      </section>
    </div>
  );
}

function ReportsPage() {
  const [searchParams] = useSearchParams();
  const { session } = useSession();
  const [reports, setReports] = useState<ReportSummary[]>([]);
  const [query, setQuery] = useState(searchParams.get('q') ?? '');
  const [sort, setSort] = useState(searchParams.get('sort') ?? 'created_at:desc');
  const [error, setError] = useState('');
  const [deletingReportId, setDeletingReportId] = useState('');
  const navigate = useNavigate();

  const load = () => {
    if (!session.logged_in) {
      setReports([]);
      return;
    }

    const params = new URLSearchParams();
    if (query.trim()) {
      params.set('q', query.trim());
    }
    if (sort) {
      const [sortBy, order] = sort.split(':');
      params.set('sort', sortBy || 'created_at');
      params.set('order', order || 'desc');
    }
    apiFetch<{ reports: ReportSummary[] }>(`/api/v1/reports?${params.toString()}`)
      .then((value) => setReports(value.reports ?? []))
      .catch((value: Error) => setError(value.message));
  };

  useEffect(() => {
    load();
  }, [query, sort, session.logged_in]);

  const openCompare = (first: string, second: string) => {
    navigate(`/reports/compare?ids=${encodeURIComponent(`${first},${second}`)}`);
  };

  const downloadReportFile = async (reportId: string, format: 'csv' | 'pdf') => {
    try {
      await downloadFile(`/api/v1/reports/${reportId}/${format}`, `report-${reportId}.${format}`);
    } catch (value) {
      setError((value as Error).message);
    }
  };

  const deleteReport = async (reportId: string) => {
    if (!window.confirm('Delete this report? This cannot be undone.')) {
      return;
    }

    setDeletingReportId(reportId);
    setError('');
    try {
      await apiFetch(`/api/v1/reports/${reportId}`, { method: 'DELETE' });
      setReports((current) => current.filter((report) => report.id !== reportId));
    } catch (value) {
      setError((value as Error).message);
    } finally {
      setDeletingReportId('');
    }
  };

  return (
    <div className="page stack">
      <div className="page__header">
        <div>
          <p className="eyebrow">My Reports</p>
          <h1 className="page__title">Saved analyses</h1>
          <p className="page__subtitle">Report cards, downloads, and the compare flow now come from the backend list route.</p>
        </div>
      </div>

      {error && <div className="card"><div className="card__body muted">{error}</div></div>}

      {!session.logged_in && (
        <section className="card">
          <div className="card__body muted">
            Saved reports are available after sign-in. <Link to="/login">Log in</Link> to access the live report list.
          </div>
        </section>
      )}

      <section className="card">
        <div className="card__body">
          <div className="toolbar">
            <div className="toolbar__group">
              <Field label="Search" style={{ minWidth: 220 }}>
                <input className="input" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search reports by title..." />
              </Field>
              <Field label="Sort" style={{ minWidth: 160 }}>
                <select className="select" value={sort} onChange={(event) => setSort(event.target.value)}>
                  <option value="created_at:desc">Newest</option>
                  <option value="created_at:asc">Oldest</option>
                  <option value="title:asc">Title A-Z</option>
                  <option value="title:desc">Title Z-A</option>
                  <option value="sku_count:desc">Most SKUs</option>
                  <option value="sku_count:asc">Fewest SKUs</option>
                </select>
              </Field>
              <button className="button button--primary button--small" type="button" onClick={load}>
                <Filter size={16} /> Apply
              </button>
            </div>
            <button className="button button--ghost button--small" type="button" onClick={() => navigate('/upload')}>
              Run Analysis
            </button>
          </div>

          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Title</th>
                  <th>SKUs</th>
                  <th>Service Level</th>
                  <th>Created</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {reports.map((report, index) => (
                  <tr key={report.id}>
                    <td>
                      <Link to={`/reports/${report.id}`} className="inline-link"><strong>{report.title}</strong></Link>
                      {!!(report.warnings?.length ?? 0) && <div className="badge badge--amber" style={{ marginTop: '0.4rem' }}>{report.warnings.length} warning{report.warnings.length > 1 ? 's' : ''}</div>}
                    </td>
                    <td>{report.sku_count}</td>
                    <td>{Math.round(report.service_level * 100)}%</td>
                    <td>{formatDate(report.created_at)}</td>
                    <td>
                      <div className="toolbar__group">
                        <button className="button button--ghost button--small" type="button" onClick={() => downloadReportFile(report.id, 'csv')}>
                          <Download size={16} /> CSV
                        </button>
                        <button className="button button--ghost button--small" type="button" onClick={() => downloadReportFile(report.id, 'pdf')}>
                          <Download size={16} /> PDF
                        </button>
                        {reports[index + 1] && (
                          <button className="button button--secondary button--small" type="button" onClick={() => openCompare(report.id, reports[index + 1].id)}>
                            Compare
                          </button>
                        )}
                        <button className="button button--danger button--small" type="button" onClick={() => deleteReport(report.id)} disabled={deletingReportId === report.id}>
                          <Trash2 size={16} /> {deletingReportId === report.id ? 'Deleting...' : 'Delete'}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!reports.length && !error && (
            <EmptyState
              icon={<BarChart3 size={40} />}
              title="No saved analyses yet"
              message="Run an analysis to save a report here and unlock compare and download actions."
              hint="Upload CSVs from the dashboard or the Upload page to create your first saved report."
              actions={<button className="button button--primary" type="button" onClick={() => navigate('/upload')}>Run your first analysis</button>}
            />
          )}
        </div>
      </section>
    </div>
  );
}

function ReportDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { session } = useSession();
  const [report, setReport] = useState<ReportDetail | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!session.logged_in || !id) {
      return;
    }

    const storedNoticeRaw = window.sessionStorage.getItem('inventory-optimizer-last-analysis-status');
    if (storedNoticeRaw) {
      try {
        const storedNotice = JSON.parse(storedNoticeRaw) as AnalysisNotice;
        if (storedNotice.reportId === id) {
          window.sessionStorage.removeItem('inventory-optimizer-last-analysis-status');
        }
      } catch {
        window.sessionStorage.removeItem('inventory-optimizer-last-analysis-status');
      }
    }

    apiFetch<ReportDetail>(`/api/v1/reports/${id}`)
      .then((value) => setReport(value))
      .catch((value: Error) => setError(value.message));
  }, [id, session.logged_in]);

  const totalAnnualCost = report?.results?.reduce((sum, result) => sum + result.simulation.avg_total_annual_cost, 0) || 0;

  return (
    <div className="page stack">
      <div className="page__header">
        <div>
          <p className="eyebrow">Report Detail</p>
          <h1 className="page__title">{report?.title || 'Saved analysis'}</h1>
          <p className="page__subtitle">Open a single analysis to inspect the full KPI set, warnings, and generated recommendations.</p>
        </div>
        <div className="toolbar__group">
          <button className="button button--ghost button--small" type="button" onClick={() => navigate('/reports')}>
            Back to reports
          </button>
          {id && <a className="button button--secondary button--small" href={`/api/v1/reports/${id}/pdf`}>PDF</a>}
        </div>
      </div>

      {error && <div className="card"><div className="card__body muted">{error}</div></div>}

      {report && (
        <section className="grid-3">
          <StatCard label="Service level" value={`${Math.round(report.service_level * 100)}%`} muted="Target fill rate" />
          <StatCard label="SKUs analyzed" value={String(report.sku_count)} muted="Catalogue entries included" />
          <StatCard label="Total annual cost" value={formatMoney(totalAnnualCost)} muted="Combined simulation cost" />
        </section>
      )}

      {report && !!report.warnings?.length && (
        <section className="card">
          <div className="card__head">
            <h2 className="card__title">Warnings</h2>
          </div>
          <div className="card__body stack">
            {report.warnings.map((warning) => (
              <div key={warning} className="badge badge--amber" style={{ width: 'fit-content' }}>{warning}</div>
            ))}
          </div>
        </section>
      )}

      {report && (
        <section className="card">
          <div className="card__head">
            <h2 className="card__title">Result Breakdown</h2>
          </div>
          <div className="card__body">
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>SKU</th>
                    <th>EOQ</th>
                    <th>ROP</th>
                    <th>Safety Stock</th>
                    <th>Annual Cost</th>
                  </tr>
                </thead>
                <tbody>
                  {report.results.map((result) => (
                    <tr key={result.parameters.sku}>
                      <td>{result.parameters.sku}</td>
                      <td>{result.policy.eoq}</td>
                      <td>{result.policy.reorder_point}</td>
                      <td>{result.policy.safety_stock}</td>
                      <td>{formatMoney(result.simulation.avg_total_annual_cost)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      )}
    </div>
  );
}

function ComparePage() {
  const [searchParams] = useSearchParams();
  const { session } = useSession();
  const ids = (searchParams.get('ids') ?? '').split(',').map((value) => value.trim()).filter(Boolean).slice(0, 2);
  const [left, setLeft] = useState<ReportDetail | null>(null);
  const [right, setRight] = useState<ReportDetail | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!session.logged_in) {
      setError('Sign in to compare saved reports.');
      return;
    }

    if (ids.length !== 2) {
      setError('Select exactly two reports to compare.');
      return;
    }

    Promise.all([
      apiFetch<ReportDetail>(`/api/v1/reports/${ids[0]}`),
      apiFetch<ReportDetail>(`/api/v1/reports/${ids[1]}`),
    ])
      .then(([a, b]) => {
        setLeft(a);
        setRight(b);
      })
      .catch((value: Error) => setError(value.message));
  }, [ids.join(','), session.logged_in]);

  const rows = useMemo(() => {
    if (!left || !right) {
      return [] as Array<{ sku: string; first: number; second: number; delta: number }>;
    }

    const map = new Map<string, { first: number; second: number }>();
    (left.results ?? []).forEach((result) => map.set(result.parameters.sku, { first: result.simulation.avg_total_annual_cost, second: 0 }));
    (right.results ?? []).forEach((result) => {
      const current = map.get(result.parameters.sku) || { first: 0, second: 0 };
      map.set(result.parameters.sku, { ...current, second: result.simulation.avg_total_annual_cost });
    });

    return Array.from(map.entries()).map(([sku, value]) => ({
      sku,
      first: value.first,
      second: value.second,
      delta: value.second - value.first,
    }));
  }, [left, right]);

  const totalFirst = left?.results?.reduce((sum, result) => sum + result.simulation.avg_total_annual_cost, 0) || 0;
  const totalSecond = right?.results?.reduce((sum, result) => sum + result.simulation.avg_total_annual_cost, 0) || 0;

  return (
    <div className="page stack">
      <div className="page__header">
        <div>
          <p className="eyebrow">Compare</p>
          <h1 className="page__title">Side-by-side report analysis</h1>
          <p className="page__subtitle">This page now pulls live report details and computes the deltas in the browser.</p>
        </div>
        <div className="toolbar__group">
          <Link className="button button--ghost button--small" to="/reports">
            Back to reports
          </Link>
        </div>
      </div>

      {error && <div className="card"><div className="card__body muted">{error}</div></div>}

      {!error && (!left || !right) && (
        <EmptyState
          icon={<BarChart3 size={40} />}
          title="Choose two reports to compare"
          message="Open the reports page and select two saved analyses to see the side-by-side comparison."
          actions={<Link className="button button--primary" to="/reports">Open reports</Link>}
        />
      )}

      {left && right && (
        <section className="grid-3">
          <StatCard label={left.title} value={formatMoney(totalFirst)} muted="Total annual cost" />
          <StatCard label={right.title} value={formatMoney(totalSecond)} muted="Total annual cost" />
          <StatCard label="Delta" value={formatMoney(totalSecond - totalFirst)} muted="Right minus left" />
        </section>
      )}

      <section className="card">
        <div className="card__body">
          <div className="compare-bars">
            {rows.map((row) => {
              const max = Math.max(row.first, row.second, 1);
              return (
                <div key={row.sku} className="compare-row">
                  <div className="compare-row__meta">
                    <strong>{row.sku}</strong>
                    <span>{formatMoney(row.delta)}</span>
                  </div>
                  <div className="compare-row__track">
                    <div className="compare-row__bar" style={{ width: `${(row.second / max) * 100}%` }} />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">Top deltas</h2>
        </div>
        <div className="card__body">
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>SKU</th>
                  <th>Left</th>
                  <th>Right</th>
                  <th>Delta</th>
                </tr>
              </thead>
              <tbody>
                {rows.slice(0, 10).map((row) => (
                  <tr key={row.sku}>
                    <td>{row.sku}</td>
                    <td>{formatMoney(row.first)}</td>
                    <td>{formatMoney(row.second)}</td>
                    <td>{formatMoney(row.delta)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </div>
  );
}

function ResultsPage() {
  const navigate = useNavigate();
  const [analysis, setAnalysis] = useState<AnalysisResponse | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    const stored = window.sessionStorage.getItem('inventory-optimizer-last-analysis');
    if (!stored) {
      setError('Run an analysis first.');
      return;
    }

    try {
      setAnalysis(JSON.parse(stored) as AnalysisResponse);
    } catch {
      setError('The last analysis payload could not be read.');
    }
  }, []);

  const results = analysis?.results ?? [];

  return (
    <div className="page stack">
      <section className="hero">
        <p className="eyebrow">Analysis complete</p>
        <h1 className="hero__title">Your inventory report is ready.</h1>
        <p className="hero__copy">
          {analysis ? `Processed ${analysis.skus_analyzed} SKU${analysis.skus_analyzed === 1 ? '' : 's'} in ${analysis.elapsed_ms}ms.` : 'No analysis payload was found in this session.'}
        </p>
        <div className="hero__actions">
          <button className="button button--primary" type="button" onClick={() => navigate('/upload')}>
            <Upload size={18} /> New analysis
          </button>
          <Link className="button button--ghost" to="/reports">
            Saved reports
          </Link>
          {analysis?.saved_report_id && (
            <Link className="button button--secondary" to={`/reports?highlight=${analysis.saved_report_id}`}>
              Open saved report
            </Link>
          )}
        </div>
      </section>

      {error && (
        <section className="card">
          <div className="card__body muted">{error}</div>
        </section>
      )}

      {analysis && (
        <section className="grid-3">
          <StatCard label="SKUs analysed" value={analysis.skus_analyzed.toString()} />
          <StatCard label="Elapsed time" value={`${analysis.elapsed_ms} ms`} />
          <StatCard label="Warnings" value={(analysis.warnings?.length ?? 0).toString()} muted={analysis.guest_locked_skus ? `${analysis.guest_locked_skus} SKU${analysis.guest_locked_skus === 1 ? '' : 's'} locked in guest mode` : 'No guest lock'} />
        </section>
      )}

      {(analysis?.warnings?.length ?? 0) ? (
        <section className="card">
          <div className="card__head">
            <h2 className="card__title">Warnings</h2>
          </div>
          <div className="card__body stack">
            {analysis?.warnings?.map((warning) => (
              <div key={warning} className="badge badge--amber" style={{ width: 'fit-content' }}>{warning}</div>
            ))}
          </div>
        </section>
      ) : null}

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">SKU insights</h2>
          <p className="card__meta">Reorder point, EOQ, safety stock, and simulation results from the live backend.</p>
        </div>
        <div className="card__body stack">
          {results.length ? results.map((item) => (
            <article key={item.parameters.sku} className="card">
              <div className="card__body">
                <div className="toolbar">
                  <div>
                    <div className="card__title">{item.parameters.sku}</div>
                    <div className="card__meta">{item.forecast.trend_label} · {item.forecast.seasonality_flag}</div>
                  </div>
                  <span className="badge badge--teal">{Math.round(item.policy.service_level * 100)}% service</span>
                </div>
                <div className="grid-3">
                  <StatCard label="Reorder point" value={Math.round(item.policy.reorder_point).toString()} />
                  <StatCard label="EOQ" value={Math.round(item.policy.eoq).toString()} />
                  <StatCard label="Safety stock" value={Math.round(item.policy.safety_stock).toString()} />
                </div>
                <div className="toolbar" style={{ marginTop: '1rem' }}>
                  <div className="toolbar__group">
                    <span className="badge badge--stone">Annual demand {Math.round(item.demand.annual_demand)}</span>
                    <span className="badge badge--stone">Stockouts {item.simulation.avg_stockouts.toFixed(1)}</span>
                    <span className="badge badge--stone">Cost {formatMoney(item.simulation.avg_total_annual_cost)}</span>
                  </div>
                </div>
              </div>
            </article>
          )) : (
            <EmptyState
              icon={<BarChart3 size={40} />}
              title="No SKU results were returned"
              message="Run a new analysis to populate this page."
              actions={<Link className="button button--primary" to="/upload">New analysis</Link>}
            />
          )}
        </div>
      </section>
    </div>
  );
}

function UploadPage() {
  const { pushToast } = useToast();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [salesFile, setSalesFile] = useState<File | null>(null);
  const [paramsFile, setParamsFile] = useState<File | null>(null);
  const [title, setTitle] = useState('');
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [analysisOverlay, setAnalysisOverlay] = useState<AnalysisOverlayState | null>(null);
  const isGuestMode = searchParams.get('mode') === 'guest';

  const fileFormatRows = {
    sales: [
      ['sku', 'week', 'units_sold'],
      ['SKU001', '2024-01-01', '12'],
      ['SKU001', '2024-01-08', '15'],
    ],
    params: [
      ['sku', 'current_inventory', 'lead_time_days', 'unit_cost', 'order_cost', 'holding_cost_rate'],
      ['SKU001', '120', '21', '8.50', '40.00', '0.25'],
    ],
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!salesFile || !paramsFile) {
      setError('Please choose both CSV files.');
      return;
    }

    setBusy(true);
    setError('');
    setResult(null);
    setAnalysisOverlay({ phase: 'loading', message: 'Running analysis and building your report...' });

    const formData = new FormData();
    formData.set('sales_file', salesFile);
    formData.set('params_file', paramsFile);
    if (title.trim()) {
      formData.set('title', title.trim());
    }

    try {
      const response = await apiFetch<AnalysisResponse>(
        '/api/v1/analyze',
        {
          method: 'POST',
          body: formData,
        },
      );
      window.sessionStorage.setItem('inventory-optimizer-last-analysis', JSON.stringify(response));
      window.sessionStorage.setItem('inventory-optimizer-last-analysis-status', JSON.stringify({
        message: `Analysis complete. Saved report ${response.saved_report_id ? response.saved_report_id : 'not saved'}. Check Dashboard > My Reports to review it.`,
        reportId: response.saved_report_id || null,
        completedAt: new Date().toISOString(),
        elapsedMs: response.elapsed_ms,
      }));
      setResult(`Analyzed ${response.skus_analyzed} SKUs in ${response.elapsed_ms}ms${response.saved_report_id ? ` and saved ${response.saved_report_id}.` : '.'}`);
      setAnalysisOverlay({
        phase: 'complete',
        message: response.saved_report_id
          ? 'Your analysis has finished. The saved report is waiting in Dashboard > My Reports.'
          : 'Your analysis has finished. Go to the Dashboard to review the results.',
        reportId: response.saved_report_id || null,
        completedAt: new Date().toISOString(),
        elapsedMs: response.elapsed_ms,
      });
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Analysis failed', description: message, tone: 'error' });
      setAnalysisOverlay(null);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="page stack">
      <AnalysisOverlay state={analysisOverlay} onContinue={() => navigate('/dashboard', { replace: true })} />
      <div className="page__header">
        <div>
          <p className="eyebrow">New analysis</p>
          <h1 className="page__title">Upload your data</h1>
          <p className="page__subtitle">This page now posts the real CSV files to <span className="badge badge--stone">/api/v1/analyze</span>.</p>
          {isGuestMode && <p className="page__subtitle"><span className="badge badge--teal">Guest mode</span> You can analyse without signing in.</p>}
        </div>
      </div>

      {(error || result) && <div className="card"><div className="card__body muted">{error || result}</div></div>}

      <section className="card">
        <div className="card__body">
          <form className="stack" onSubmit={submit}>
            <Field label="Analysis Title">
              <input className="input" value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Q2 inventory review" />
            </Field>
            <div className="grid-2">
              <Field label="Sales history CSV">
                <input className="input" type="file" accept=".csv" onChange={(event) => setSalesFile(event.target.files?.[0] || null)} />
              </Field>
              <Field label="SKU parameters CSV">
                <input className="input" type="file" accept=".csv" onChange={(event) => setParamsFile(event.target.files?.[0] || null)} />
              </Field>
            </div>
            <button className="button button--primary" type="submit" disabled={busy}>
              <Upload size={18} /> {busy ? 'Running analysis...' : 'Run analysis'}
            </button>
          </form>
        </div>
      </section>

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">Expected File Formats</h2>
          <p className="card__meta">Upload the CSV structures shown below to keep the analysis pipeline aligned with the backend parser.</p>
        </div>
        <div className="card__body">
          <div className="grid-2">
            <article className="format-card">
              <div className="toolbar">
                <div>
                  <div className="stat__label">Sales history</div>
                  <h3 className="card__title">sales_history.csv</h3>
                </div>
                <span className="badge badge--teal">CSV</span>
              </div>
              <div className="table-wrap">
                <table className="table format-table">
                  <thead>
                    <tr>
                      {fileFormatRows.sales[0].map((header) => <th key={header}>{header}</th>)}
                    </tr>
                  </thead>
                  <tbody>
                    {fileFormatRows.sales.slice(1).map((row) => (
                      <tr key={row.join('-')}>
                        {row.map((cell) => <td key={cell}>{cell}</td>)}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <p className="muted">One row per SKU per week. Dates as YYYY-MM-DD.</p>
            </article>

            <article className="format-card">
              <div className="toolbar">
                <div>
                  <div className="stat__label">SKU parameters</div>
                  <h3 className="card__title">sku_parameters.csv</h3>
                </div>
                <span className="badge badge--amber">CSV</span>
              </div>
              <div className="table-wrap">
                <table className="table format-table format-table--wide">
                  <thead>
                    <tr>
                      {fileFormatRows.params[0].map((header) => <th key={header}>{header}</th>)}
                    </tr>
                  </thead>
                  <tbody>
                    {fileFormatRows.params.slice(1).map((row) => (
                      <tr key={row.join('-')}>
                        {row.map((cell) => <td key={cell}>{cell}</td>)}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <p className="muted">One row per SKU. Holding cost rate is a decimal, for example 0.25 = 25%.</p>
            </article>
          </div>
        </div>
      </section>
    </div>
  );
}

function AuthPage({ mode }: { mode: 'login' | 'signup' }) {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { refresh, session } = useSession();
  const { pushToast } = useToast();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [signupStep, setSignupStep] = useState<1 | 2>(1);
  const [preferredCurrency, setPreferredCurrency] = useState('NGN');
  const [countryCode, setCountryCode] = useState('NG');
  const [businessType, setBusinessType] = useState('retail');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const redirect = searchParams.get('redirect') || '/';

  if (session.logged_in) {
    return <Navigate to={redirect} replace />;
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault();

    if (mode === 'signup' && signupStep === 1) {
      if (!email.trim() || !password || !confirmPassword) {
        setError('Enter your account details to continue.');
        return;
      }
      if (password !== confirmPassword) {
        setError('Passwords do not match.');
        return;
      }
      setError('');
      setSignupStep(2);
      return;
    }

    setBusy(true);
    setError('');

    try {
      const path = mode === 'login' ? '/api/v1/auth/login' : '/api/v1/auth/register';
      const body = mode === 'login'
        ? { email, password }
        : {
          email,
          password,
          confirm_password: confirmPassword,
          preferred_currency: preferredCurrency,
          country_code: countryCode,
          business_type: businessType,
        };

      const response = await apiFetch<{ access_token: string } & { refresh_token?: string }>(path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if ('access_token' in response) {
        window.localStorage.setItem(TOKEN_KEY, response.access_token);
      }

      await refresh();
      pushToast({
        title: mode === 'login' ? 'Signed in' : 'Account created',
        description: 'Your session is active and ready for live routes.',
        tone: 'success',
      });
      navigate(redirect, { replace: true });
    } catch (value) {
      const message = (value as Error).message;
      setError(message);
      pushToast({ title: 'Authentication failed', description: message, tone: 'error' });
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="page">
      <section className="card" style={{ maxWidth: 560, margin: '0 auto' }}>
        <div className="card__body">
          <p className="eyebrow">{mode === 'login' ? 'Welcome back' : 'Create account'}</p>
          <h1 className="page__title" style={{ fontSize: '2rem' }}>{mode === 'login' ? 'Sign in' : 'Sign up'}</h1>
          <p className="page__subtitle">
            The session is stored as a bearer token so the SPA can use the live API routes.
            {mode === 'signup' && ' New accounts now include a short business setup step for currency, country, and business type.'}
          </p>

          {mode === 'signup' && (
            <div className="toolbar" style={{ marginBottom: '1rem' }}>
              <span className={`badge ${signupStep === 1 ? 'badge--teal' : 'badge--stone'}`}>1. Account details</span>
              <span className={`badge ${signupStep === 2 ? 'badge--teal' : 'badge--stone'}`}>2. Business profile</span>
            </div>
          )}

          <form className="stack" onSubmit={submit}>
            {mode === 'login' || signupStep === 1 ? (
              <>
                <Field label="Email">
                  <input className="input" type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
                </Field>
                <Field label="Password">
                  <input className="input" type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
                </Field>
                {mode === 'signup' && (
                  <Field label="Confirm password">
                    <input className="input" type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} required />
                  </Field>
                )}
              </>
            ) : (
              <>
                <Field label="Preferred currency">
                  <select className="input" value={preferredCurrency} onChange={(event) => setPreferredCurrency(event.target.value)} required>
                    {CURRENCY_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </Field>
                <Field label="Country">
                  <select className="input" value={countryCode} onChange={(event) => setCountryCode(event.target.value)} required>
                    {COUNTRY_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </Field>
                <Field label="Business type">
                  <select className="input" value={businessType} onChange={(event) => setBusinessType(event.target.value)} required>
                    {BUSINESS_TYPE_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </Field>
                <div className="card" style={{ margin: 0, background: 'linear-gradient(135deg, rgba(245, 238, 228, 0.7), rgba(251, 248, 241, 0.95))' }}>
                  <div className="card__body stack">
                    <div className="card__title" style={{ fontSize: '1rem' }}>Why this matters</div>
                    <p className="muted">Your currency controls how costs are displayed, your country helps us shape regional defaults later, and your business type helps tailor replenishment logic.</p>
                  </div>
                </div>
              </>
            )}
            {error && <div className="muted">{error}</div>}
            <div className="toolbar__group">
              {mode === 'signup' && signupStep === 2 && (
                <button className="button button--ghost" type="button" onClick={() => setSignupStep(1)} disabled={busy}>
                  Back
                </button>
              )}
              <button className="button button--primary" type="submit" disabled={busy}>
                {busy ? 'Working...' : mode === 'login' ? 'Login' : signupStep === 1 ? 'Next' : 'Create account'}
              </button>
            </div>
          </form>
        </div>
      </section>
    </div>
  );
}

function SubscriptionPage() {
  const { session } = useSession();
  const checkoutUrl = BILLING_CHECKOUT_URL.trim();
  const recommendedPlatform = 'Paddle';
  const [notificationSettings, setNotificationSettings] = useState<NotificationSettings>({
    user_id: '',
    enabled: false,
    frequency: 'daily',
    scheduled_time: '09:00',
    email_override: '',
    timezone: 'UTC',
    updated_at: '',
  });
  const [settingsStatus, setSettingsStatus] = useState('');
  const [settingsLoading, setSettingsLoading] = useState(false);

  useEffect(() => {
    if (!session.logged_in || (session.account_status !== 'trial' && session.account_status !== 'premium')) {
      return;
    }

    let mounted = true;
    setSettingsLoading(true);
    apiFetch<NotificationSettings>('/api/v1/notification-settings')
      .then((value) => {
        if (mounted) {
          setNotificationSettings({
            user_id: value.user_id,
            enabled: value.enabled,
            frequency: value.frequency || 'daily',
            scheduled_time: value.scheduled_time || '09:00',
            email_override: value.email_override || '',
            timezone: value.timezone || 'UTC',
            last_sent_at: value.last_sent_at || null,
            updated_at: value.updated_at || '',
          });
        }
      })
      .catch((value) => {
        if (mounted) {
          setSettingsStatus((value as Error).message);
        }
      })
      .finally(() => {
        if (mounted) {
          setSettingsLoading(false);
        }
      });

    return () => {
      mounted = false;
    };
  }, [session.account_status, session.logged_in]);

  const saveNotificationSettings = async (event: FormEvent) => {
    event.preventDefault();
    setSettingsLoading(true);
    setSettingsStatus('');

    try {
      const value = await apiFetch<NotificationSettings>('/api/v1/notification-settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled: notificationSettings.enabled,
          frequency: notificationSettings.frequency,
          scheduled_time: notificationSettings.scheduled_time,
          email_override: notificationSettings.email_override,
        }),
      });

      setNotificationSettings({
        user_id: value.user_id,
        enabled: value.enabled,
        frequency: value.frequency || 'daily',
        scheduled_time: value.scheduled_time || '09:00',
        email_override: value.email_override || '',
        timezone: value.timezone || 'UTC',
        last_sent_at: value.last_sent_at || null,
        updated_at: value.updated_at || '',
      });
      setSettingsStatus('Notification schedule saved.');
    } catch (value) {
      setSettingsStatus((value as Error).message);
    } finally {
      setSettingsLoading(false);
    }
  };

  return (
    <div className="page stack">
      <section className="billing-hero card">
        <div className="card__body stack">
          <p className="eyebrow">Subscription</p>
          <h1 className="page__title">Unlock premium access after your 6-month trial</h1>
          <p className="page__subtitle">
            Trial access is automatic for new accounts. When it ends, premium routes stop working until you subscribe.
          </p>

          <div className="billing-hero__actions">
            {checkoutUrl ? (
              <a className="button button--primary" href={checkoutUrl} target="_blank" rel="noreferrer">
                Continue to checkout
              </a>
            ) : (
              <button className="button button--primary" type="button" disabled>
                Connect checkout URL
              </button>
            )}
            <Link className="button button--secondary" to={session.logged_in ? '/dashboard' : '/signup'}>
              {session.logged_in ? 'Back to app' : 'Create account'}
            </Link>
          </div>
        </div>
      </section>

      <section className="grid-2">
        <article className="card">
          <div className="card__body stack">
            <div>
              <p className="eyebrow">Recommended platform</p>
              <h2 className="card__title">{recommendedPlatform}</h2>
            </div>
            <p className="muted">
              Best fit for a global SaaS subscription because it uses a hosted checkout, minimizes PCI scope, and can act as merchant of record for tax handling and compliance in many regions.
            </p>
            <ul className="feature-list">
              <li>Hosted checkout reduces payment security risk.</li>
              <li>Global subscription support with localized payment methods.</li>
              <li>Lower compliance burden than building payments yourself.</li>
            </ul>
          </div>
        </article>

        <article className="card">
          <div className="card__body stack">
            <div>
              <p className="eyebrow">What premium includes</p>
              <h2 className="card__title">All advanced features</h2>
            </div>
            <div className="feature-grid">
              <div className="feature-pill">Saved reports</div>
              <div className="feature-pill">Full catalogue analytics</div>
              <div className="feature-pill">Smart Records generation</div>
              <div className="feature-pill">PDF and CSV downloads</div>
              <div className="feature-pill">Budget optimization</div>
              <div className="feature-pill">Report compare</div>
            </div>
            <p className="muted">If your trial expires, these routes are blocked until the subscription is active again.</p>
          </div>
        </article>
      </section>

      {session.logged_in && (session.account_status === 'trial' || session.account_status === 'premium') && (
        <section className="card">
          <div className="card__body stack">
            <div className="toolbar">
              <div>
                <p className="eyebrow">Schedule settings</p>
                <h2 className="card__title">Premium replenishment notifications</h2>
              </div>
              <span className="badge badge--teal">Premium only</span>
            </div>
            <p className="muted">Choose when the backend job runner should email your replenishment alerts and keep the notification center in sync.</p>

            <form className="stack" onSubmit={saveNotificationSettings}>
              <label className="field" style={{ width: 'fit-content', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                <span className="field__label">Enable alert emails</span>
                <input
                  type="checkbox"
                  checked={notificationSettings.enabled}
                  onChange={(event) => setNotificationSettings((current) => ({ ...current, enabled: event.target.checked }))}
                />
              </label>

              <div className="grid-2">
                <Field label="Frequency">
                  <select
                    className="input"
                    value={notificationSettings.frequency}
                    onChange={(event) => setNotificationSettings((current) => ({ ...current, frequency: event.target.value as 'daily' | 'weekly' }))}
                  >
                    <option value="daily">Daily</option>
                    <option value="weekly">Weekly</option>
                  </select>
                </Field>
                <Field label="Send at">
                  <input
                    className="input"
                    type="time"
                    value={notificationSettings.scheduled_time}
                    onChange={(event) => setNotificationSettings((current) => ({ ...current, scheduled_time: event.target.value }))}
                  />
                </Field>
              </div>

              <Field label="Notification email">
                <input
                  className="input"
                  type="email"
                  value={notificationSettings.email_override}
                  onChange={(event) => setNotificationSettings((current) => ({ ...current, email_override: event.target.value }))}
                  placeholder={session.user_email || 'leave blank to use your account email'}
                />
              </Field>

              <div className="toolbar__group">
                <button className="button button--primary" type="submit" disabled={settingsLoading}>
                  {settingsLoading ? 'Saving...' : 'Save notification schedule'}
                </button>
                <span className="muted">{settingsStatus || 'Digest emails are driven by the backend worker.'}</span>
              </div>
            </form>
          </div>
        </section>
      )}

      {session.logged_in && session.account_status !== 'trial' && session.account_status !== 'premium' && (
        <section className="card">
          <div className="card__body muted">
            Notification scheduling is available on premium and trial accounts.
          </div>
        </section>
      )}
    </div>
  );
}

function StatCard({ label, value, muted }: { label: string; value: string; muted?: string }) {
  return (
    <article className="card stat">
      <div className="card__body">
        <div className="stat__label">{label}</div>
        <p className="stat__value">{value}</p>
        {muted && <div className="stat__muted">{muted}</div>}
      </div>
    </article>
  );
}

function Field({ label, children, style }: { label: string; children: ReactNode; style?: React.CSSProperties }) {
  return (
    <label className="field" style={style}>
      <span className="field__label">{label}</span>
      {children}
    </label>
  );
}

function Footer() {
  return (
    <footer className="footer">
      <div className="footer__inner">
        Inventory Optimizer · React frontend wired to live backend routes.
      </div>
    </footer>
  );
}

export default App;