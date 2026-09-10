import { useState, useEffect, useCallback } from 'react';
import { fetchCars, fetchImportStatus } from './api';
import CarsTable from './CarsTable';
import ImportButton from './ImportButton';
import Stats from './Stats';
import './App.css';

const PAGE_SIZE = 20;

export default function App() {
  const [cars, setCars] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastImport, setLastImport] = useState(null);
  const [importMessage, setImportMessage] = useState(null);
  const [importError, setImportError] = useState(null);

  const reloadData = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const [carsData, statusData] = await Promise.all([
        fetchCars(page, PAGE_SIZE),
        fetchImportStatus(),
      ]);

      setCars(Array.isArray(carsData.items) ? carsData.items : []);
      setTotal(typeof carsData.total === 'number' ? carsData.total : 0);
      setLastImport(statusData.last_import || null);
    } catch (err) {
      setError(err.status ? `${err.status} ${err.message}` : err.message);
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    reloadData();
  }, [reloadData]);

  function handlePageChange(newPage) {
    setPage(newPage);
  }

  function handleImportSuccess(message) {
    setImportMessage(message);
    setImportError(null);
    setPage(1);
    reloadData();
  }

  function handleImportError(err) {
    setImportError(err.status ? `${err.status} ${err.message}` : err.message);
    setImportMessage(null);
  }

  return (
    <div className="app">
      <header className="app-header">
        <h1>Autocheck.kz</h1>
        <p className="subtitle">Car inventory</p>
      </header>

      <Stats total={total} lastImport={lastImport} />

      <div className="toolbar">
        <ImportButton
          onSuccess={handleImportSuccess}
          onError={handleImportError}
        />
      </div>

      {importMessage && (
        <div className="banner banner-success" role="status">
          {importMessage}
        </div>
      )}
      {importError && (
        <div className="banner banner-error" role="alert">
          {importError}
        </div>
      )}
      {error && (
        <div className="banner banner-error" role="alert">
          Failed to load cars: {error}
        </div>
      )}

      {loading ? (
        <div className="loading" aria-label="Loading">
          <span className="spinner" />
          Loading...
        </div>
      ) : (
        <CarsTable
          cars={cars}
          page={page}
          pageSize={PAGE_SIZE}
          total={total}
          onPageChange={handlePageChange}
        />
      )}
    </div>
  );
}
