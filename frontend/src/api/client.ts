const API_BASE = ''

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  // Only set Content-Type for non-FormData bodies
  if (options.body && !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  if (res.status === 401 && !path.includes('/api/auth/login')) {
    localStorage.removeItem('token')
    window.location.href = '/admin/login'
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const error = await res.text()
    throw new Error(error || res.statusText)
  }

  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return undefined as T
  }

  return res.json()
}
