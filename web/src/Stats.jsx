import { fetchImportStatus } from './api';

export function formatDateTime(iso) {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  const months = [
    'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
    'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
  ];
  const day = String(d.getDate()).padStart(2, '0');
  const month = months[d.getMonth()];
  const year = d.getFullYear();
  const hours = String(d.getHours()).padStart(2, '0');
  const mins = String(d.getMinutes()).padStart(2, '0');
  return `${day} ${month} ${year} ${hours}:${mins}`;
}

export default function Stats({ total, lastImport }) {
  const formatted = formatDateTime(lastImport?.finished_at || lastImport?.started_at);
  const lastImportText = formatted || 'never';

  return (
    <div className="stats">
      <div className="stat-item">
        <span className="stat-label">Total cars:</span>
        <span className="stat-value">{total ?? 0}</span>
      </div>
      <div className="stat-item">
        <span className="stat-label">Last import:</span>
        <span className="stat-value">{lastImportText}</span>
      </div>
    </div>
  );
}
