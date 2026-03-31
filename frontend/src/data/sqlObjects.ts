import type { SQLView, SQLFunction, SQLTrigger, SQLSequence, SQLEnum } from '../types/database'

export const sqlViews: SQLView[] = [
  {
    name: 'active_users',
    schema: 'public',
    materialized: false,
    definition: `CREATE OR REPLACE VIEW public.active_users AS
SELECT u.id, u.email, u.username, u.role,
       u.last_login_at, u.created_at,
       COUNT(o.id) AS order_count,
       COALESCE(SUM(o.total_amount), 0) AS total_spent
FROM public.users u
LEFT JOIN public.orders o ON o.user_id = u.id
WHERE u.is_active = true
  AND u.last_login_at >= NOW() - INTERVAL '30 days'
GROUP BY u.id;`,
    columns: [
      { name: 'id', type: 'uuid', nullable: false, primaryKey: false, defaultValue: null },
      { name: 'email', type: 'varchar(255)', nullable: false, primaryKey: false, defaultValue: null },
      { name: 'username', type: 'varchar(100)', nullable: false, primaryKey: false, defaultValue: null },
      { name: 'role', type: 'varchar(20)', nullable: false, primaryKey: false, defaultValue: null },
      { name: 'last_login_at', type: 'timestamptz', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'created_at', type: 'timestamptz', nullable: false, primaryKey: false, defaultValue: null },
      { name: 'order_count', type: 'bigint', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'total_spent', type: 'numeric', nullable: true, primaryKey: false, defaultValue: null },
    ],
  },
  {
    name: 'order_summary',
    schema: 'public',
    materialized: true,
    definition: `CREATE MATERIALIZED VIEW public.order_summary AS
SELECT DATE_TRUNC('day', o.created_at) AS day,
       COUNT(*) AS order_count,
       SUM(o.total_amount) AS revenue,
       AVG(o.total_amount) AS avg_order_value,
       COUNT(DISTINCT o.user_id) AS unique_customers
FROM public.orders o
WHERE o.status != 'cancelled'
GROUP BY day
ORDER BY day DESC;`,
    columns: [
      { name: 'day', type: 'timestamptz', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'order_count', type: 'bigint', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'revenue', type: 'numeric', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'avg_order_value', type: 'numeric', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'unique_customers', type: 'bigint', nullable: true, primaryKey: false, defaultValue: null },
    ],
  },
  {
    name: 'daily_page_views',
    schema: 'analytics',
    materialized: true,
    definition: `CREATE MATERIALIZED VIEW analytics.daily_page_views AS
SELECT DATE_TRUNC('day', created_at) AS day,
       path,
       COUNT(*) AS views,
       COUNT(DISTINCT user_id) AS unique_visitors,
       AVG(duration_ms) AS avg_duration_ms
FROM analytics.page_views
GROUP BY day, path
ORDER BY day DESC, views DESC;`,
    columns: [
      { name: 'day', type: 'timestamptz', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'path', type: 'text', nullable: false, primaryKey: false, defaultValue: null },
      { name: 'views', type: 'bigint', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'unique_visitors', type: 'bigint', nullable: true, primaryKey: false, defaultValue: null },
      { name: 'avg_duration_ms', type: 'numeric', nullable: true, primaryKey: false, defaultValue: null },
    ],
  },
]

export const sqlFunctions: SQLFunction[] = [
  {
    name: 'update_updated_at',
    schema: 'public',
    language: 'plpgsql',
    returnType: 'trigger',
    args: [],
    volatility: 'VOLATILE',
    definition: `CREATE OR REPLACE FUNCTION public.update_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$;`,
  },
  {
    name: 'calculate_order_total',
    schema: 'public',
    language: 'plpgsql',
    returnType: 'numeric',
    args: [
      { name: 'p_order_id', type: 'bigint', mode: 'IN' },
    ],
    volatility: 'STABLE',
    definition: `CREATE OR REPLACE FUNCTION public.calculate_order_total(p_order_id bigint)
RETURNS numeric
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
  v_total numeric;
BEGIN
  SELECT COALESCE(SUM(oi.quantity * oi.unit_price), 0)
  INTO v_total
  FROM public.order_items oi
  WHERE oi.order_id = p_order_id;

  RETURN v_total;
END;
$$;`,
  },
  {
    name: 'search_users',
    schema: 'public',
    language: 'sql',
    returnType: 'SETOF users',
    args: [
      { name: 'search_term', type: 'text', mode: 'IN' },
      { name: 'max_results', type: 'integer', mode: 'IN' },
    ],
    volatility: 'STABLE',
    definition: `CREATE OR REPLACE FUNCTION public.search_users(search_term text, max_results integer DEFAULT 20)
RETURNS SETOF public.users
LANGUAGE sql
STABLE
AS $$
  SELECT *
  FROM public.users
  WHERE username ILIKE '%' || search_term || '%'
     OR email ILIKE '%' || search_term || '%'
  ORDER BY last_login_at DESC NULLS LAST
  LIMIT max_results;
$$;`,
  },
  {
    name: 'check_permission',
    schema: 'auth',
    language: 'plpgsql',
    returnType: 'boolean',
    args: [
      { name: 'p_role_id', type: 'integer', mode: 'IN' },
      { name: 'p_resource', type: 'text', mode: 'IN' },
      { name: 'p_action', type: 'text', mode: 'IN' },
    ],
    volatility: 'STABLE',
    definition: `CREATE OR REPLACE FUNCTION auth.check_permission(
  p_role_id integer, p_resource text, p_action text
)
RETURNS boolean
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
  RETURN EXISTS (
    SELECT 1 FROM auth.permissions
    WHERE role_id = p_role_id
      AND resource = p_resource
      AND action = p_action
  );
END;
$$;`,
  },
  {
    name: 'track_event',
    schema: 'analytics',
    language: 'plpgsql',
    returnType: 'bigint',
    args: [
      { name: 'p_event_type', type: 'varchar(100)', mode: 'IN' },
      { name: 'p_payload', type: 'jsonb', mode: 'IN' },
      { name: 'p_user_id', type: 'uuid', mode: 'IN' },
    ],
    volatility: 'VOLATILE',
    definition: `CREATE OR REPLACE FUNCTION analytics.track_event(
  p_event_type varchar(100), p_payload jsonb DEFAULT NULL, p_user_id uuid DEFAULT NULL
)
RETURNS bigint
LANGUAGE plpgsql
AS $$
DECLARE
  v_id bigint;
BEGIN
  INSERT INTO analytics.events (event_type, payload, user_id)
  VALUES (p_event_type, p_payload, p_user_id)
  RETURNING id INTO v_id;

  RETURN v_id;
END;
$$;`,
  },
]

export const sqlTriggers: SQLTrigger[] = [
  {
    name: 'trg_users_updated_at',
    schema: 'public',
    table: 'users',
    event: 'UPDATE',
    timing: 'BEFORE',
    forEach: 'ROW',
    function: 'public.update_updated_at()',
    enabled: true,
    definition: `CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON public.users
  FOR EACH ROW
  EXECUTE FUNCTION public.update_updated_at();`,
  },
  {
    name: 'trg_orders_updated_at',
    schema: 'public',
    table: 'orders',
    event: 'UPDATE',
    timing: 'BEFORE',
    forEach: 'ROW',
    function: 'public.update_updated_at()',
    enabled: true,
    definition: `CREATE TRIGGER trg_orders_updated_at
  BEFORE UPDATE ON public.orders
  FOR EACH ROW
  EXECUTE FUNCTION public.update_updated_at();`,
  },
  {
    name: 'trg_orders_recalculate_total',
    schema: 'public',
    table: 'order_items',
    event: 'INSERT OR UPDATE OR DELETE',
    timing: 'AFTER',
    forEach: 'ROW',
    function: 'public.recalculate_order_total()',
    enabled: true,
    definition: `CREATE TRIGGER trg_orders_recalculate_total
  AFTER INSERT OR UPDATE OR DELETE ON public.order_items
  FOR EACH ROW
  EXECUTE FUNCTION public.recalculate_order_total();`,
  },
  {
    name: 'trg_audit_users',
    schema: 'public',
    table: 'users',
    event: 'INSERT OR UPDATE OR DELETE',
    timing: 'AFTER',
    forEach: 'ROW',
    function: 'public.audit_log()',
    enabled: false,
    definition: `CREATE TRIGGER trg_audit_users
  AFTER INSERT OR UPDATE OR DELETE ON public.users
  FOR EACH ROW
  EXECUTE FUNCTION public.audit_log();`,
  },
]

export const sqlSequences: SQLSequence[] = [
  { name: 'orders_id_seq', schema: 'public', dataType: 'bigint', startValue: 1, increment: 1, minValue: 1, maxValue: 9223372036854775807, currentValue: 1847291, cacheSize: 1, cycle: false, ownedBy: 'public.orders.id' },
  { name: 'products_id_seq', schema: 'public', dataType: 'integer', startValue: 1, increment: 1, minValue: 1, maxValue: 2147483647, currentValue: 12847, cacheSize: 1, cycle: false, ownedBy: 'public.products.id' },
  { name: 'order_items_id_seq', schema: 'public', dataType: 'bigint', startValue: 1, increment: 1, minValue: 1, maxValue: 9223372036854775807, currentValue: 4219847, cacheSize: 10, cycle: false, ownedBy: 'public.order_items.id' },
  { name: 'categories_id_seq', schema: 'public', dataType: 'integer', startValue: 1, increment: 1, minValue: 1, maxValue: 2147483647, currentValue: 142, cacheSize: 1, cycle: false, ownedBy: 'public.categories.id' },
  { name: 'addresses_id_seq', schema: 'public', dataType: 'integer', startValue: 1, increment: 1, minValue: 1, maxValue: 2147483647, currentValue: 341201, cacheSize: 1, cycle: false, ownedBy: 'public.addresses.id' },
  { name: 'page_views_id_seq', schema: 'analytics', dataType: 'bigint', startValue: 1, increment: 1, minValue: 1, maxValue: 9223372036854775807, currentValue: 28491042, cacheSize: 50, cycle: false, ownedBy: 'analytics.page_views.id' },
  { name: 'events_id_seq', schema: 'analytics', dataType: 'bigint', startValue: 1, increment: 1, minValue: 1, maxValue: 9223372036854775807, currentValue: 5829104, cacheSize: 20, cycle: false, ownedBy: 'analytics.events.id' },
  { name: 'roles_id_seq', schema: 'auth', dataType: 'integer', startValue: 1, increment: 1, minValue: 1, maxValue: 2147483647, currentValue: 8, cacheSize: 1, cycle: false, ownedBy: 'auth.roles.id' },
]

export const sqlEnums: SQLEnum[] = [
  { name: 'order_status', schema: 'public', values: ['pending', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'] },
  { name: 'user_role', schema: 'public', values: ['admin', 'user', 'viewer', 'moderator'] },
  { name: 'payment_method', schema: 'public', values: ['credit_card', 'debit_card', 'paypal', 'bank_transfer', 'crypto'] },
  { name: 'event_type', schema: 'analytics', values: ['page_view', 'click', 'form_submit', 'purchase', 'signup', 'logout'] },
]
