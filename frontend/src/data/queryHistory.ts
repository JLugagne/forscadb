export interface QueryHistoryEntry {
  id: string
  query: string
  executedAt: string
  duration: number
  rowCount: number
  status: 'success' | 'error'
  error?: string
}

export const queryHistory: QueryHistoryEntry[] = [
  {
    id: 'qh-1',
    query: `SELECT id, email, username, role, is_active, last_login_at, created_at
FROM public.users
WHERE is_active = true
ORDER BY created_at DESC
LIMIT 50;`,
    executedAt: '2026-03-29T10:22:14Z',
    duration: 23.4,
    rowCount: 284392,
    status: 'success',
  },
  {
    id: 'qh-2',
    query: `SELECT u.username, COUNT(o.id) AS order_count, SUM(o.total_amount) AS total_spent
FROM public.users u
JOIN public.orders o ON o.user_id = u.id
WHERE o.status = 'completed'
GROUP BY u.username
ORDER BY total_spent DESC
LIMIT 20;`,
    executedAt: '2026-03-29T10:18:42Z',
    duration: 187.2,
    rowCount: 20,
    status: 'success',
  },
  {
    id: 'qh-3',
    query: `UPDATE public.users
SET role = 'admin'
WHERE email = 'grace@company.com'
RETURNING id, email, role;`,
    executedAt: '2026-03-29T10:15:01Z',
    duration: 4.1,
    rowCount: 1,
    status: 'success',
  },
  {
    id: 'qh-4',
    query: `SELECT p.name, p.sku, p.price, c.name AS category
FROM public.products p
LEFT JOIN public.categories c ON c.id = p.category_id
WHERE p.stock_quantity < 10 AND p.is_active = true
ORDER BY p.stock_quantity ASC;`,
    executedAt: '2026-03-29T10:10:33Z',
    duration: 45.7,
    rowCount: 127,
    status: 'success',
  },
  {
    id: 'qh-5',
    query: `SELECT DATE_TRUNC('day', created_at) AS day,
       COUNT(*) AS orders,
       SUM(total_amount) AS revenue
FROM public.orders
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY day
ORDER BY day DESC;`,
    executedAt: '2026-03-29T09:58:12Z',
    duration: 312.8,
    rowCount: 30,
    status: 'success',
  },
  {
    id: 'qh-6',
    query: `SELECT * FROM public.user_preferences WHERE user_id = 'abc';`,
    executedAt: '2026-03-29T09:52:44Z',
    duration: 1.2,
    rowCount: 0,
    status: 'error',
    error: 'ERROR: relation "public.user_preferences" does not exist',
  },
  {
    id: 'qh-7',
    query: `DELETE FROM public.sessions
WHERE expires_at < NOW();`,
    executedAt: '2026-03-29T09:45:00Z',
    duration: 89.3,
    rowCount: 12847,
    status: 'success',
  },
  {
    id: 'qh-8',
    query: `SELECT oi.product_id, p.name, SUM(oi.quantity) AS total_sold
FROM public.order_items oi
JOIN public.products p ON p.id = oi.product_id
JOIN public.orders o ON o.id = oi.order_id
WHERE o.created_at >= '2026-01-01'
GROUP BY oi.product_id, p.name
ORDER BY total_sold DESC
LIMIT 10;`,
    executedAt: '2026-03-29T09:30:15Z',
    duration: 542.1,
    rowCount: 10,
    status: 'success',
  },
  {
    id: 'qh-9',
    query: `EXPLAIN ANALYZE
SELECT u.id, u.email, a.city, a.country
FROM public.users u
JOIN public.addresses a ON a.user_id = u.id
WHERE a.country = 'FR';`,
    executedAt: '2026-03-28T17:42:30Z',
    duration: 15.6,
    rowCount: 8924,
    status: 'success',
  },
  {
    id: 'qh-10',
    query: `INSERT INTO public.categories (name, slug, parent_id)
VALUES ('Electronics', 'electronics', NULL)
RETURNING id, name, slug;`,
    executedAt: '2026-03-28T16:20:00Z',
    duration: 2.8,
    rowCount: 1,
    status: 'success',
  },
]
