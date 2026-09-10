function fmtInt(n) {
  if (n == null || Number.isNaN(n)) return '—';
  return Number(n).toLocaleString('en-US');
}

function fmtDefects(defects) {
  if (!Array.isArray(defects) || defects.length === 0) return '—';
  return defects.join(', ');
}

export default function CarsTable({ cars, page, pageSize, total, onPageChange }) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  if (page > totalPages) {
    // safety: if backend total changed, clamp page
    onPageChange(totalPages);
    return null;
  }

  return (
    <div className="table-wrap">
      <table className="cars-table">
        <thead>
          <tr>
            <th>VIN</th>
            <th>Brand</th>
            <th>Model</th>
            <th>Year</th>
            <th>Mileage</th>
            <th>Price</th>
            <th>Defects</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {cars.length === 0 ? (
            <tr>
              <td colSpan={8} className="empty-row">No cars imported yet</td>
            </tr>
          ) : (
            cars.map((car) => (
              <tr key={car.vin}>
                <td className="cell-vin">{car.vin}</td>
                <td>{car.brand || '—'}</td>
                <td>{car.model || '—'}</td>
                <td>{car.year != null ? car.year : '—'}</td>
                <td>{car.mileage_km != null ? `${fmtInt(car.mileage_km)} km` : '—'}</td>
                <td>{car.price != null ? `${fmtInt(car.price)} ₸` : '—'}</td>
                <td>{fmtDefects(car.defects)}</td>
                <td>
                  <span className={`badge badge-${car.status || 'active'}`}>
                    {car.status || 'active'}
                  </span>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>

      <div className="pagination">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          Prev
        </button>
        <span className="page-indicator">
          Page {page} of {totalPages}
        </span>
        <button
          type="button"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </button>
      </div>
    </div>
  );
}
