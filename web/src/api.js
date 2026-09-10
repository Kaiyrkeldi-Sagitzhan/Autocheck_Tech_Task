const API_BASE = '/api';

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = await res.json();
      detail = body.error || body.message || detail;
    } catch {
      // keep default status text
    }
    const err = new Error(detail);
    err.status = res.status;
    throw err;
  }

  if (res.status === 204) return null;
  return res.json();
}

export function fetchCars(page = 1, pageSize = 20) {
  const params = new URLSearchParams({ page: String(page), limit: String(pageSize) });
  return request(`/cars?${params.toString()}`);
}

export function fetchImportStatus() {
  return request('/import/status');
}

export function uploadImport(file) {
  const form = new FormData();
  form.append('file', file);
  return fetch('/api/import', {
    method: 'POST',
    body: form,
  }).then(async (res) => {
    if (!res.ok) {
      let detail = res.statusText;
      try {
        const body = await res.json();
        detail = body.error || body.message || detail;
      } catch {
        // keep default
      }
      const err = new Error(detail);
      err.status = res.status;
      throw err;
    }
    return res.json();
  });
}
