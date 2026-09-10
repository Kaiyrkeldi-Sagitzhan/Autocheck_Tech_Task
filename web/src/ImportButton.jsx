import { useRef } from 'react';
import { uploadImport } from './api';

export default function ImportButton({ onSuccess, onError }) {
  const inputRef = useRef(null);
  const importingRef = useRef(false);

  async function handleChange(e) {
    const file = e.target.files?.[0];
    if (!file) return;

    // Reset input so the same file can be selected again if needed
    e.target.value = '';

    if (importingRef.current) return;
    importingRef.current = true;

    try {
      const report = await uploadImport(file);
      const summary = `Imported: ${report.created} created, ${report.updated} updated, ${report.skipped} skipped, ${report.errors?.length || 0} failed`;
      onSuccess?.(summary);
    } catch (err) {
      onError?.(err);
    } finally {
      importingRef.current = false;
    }
  }

  return (
    <div className="import-section">
      <input
        ref={inputRef}
        type="file"
        accept=".csv"
        onChange={handleChange}
        className="file-input"
      />
      <button
        type="button"
        className="import-btn"
        onClick={() => inputRef.current?.click()}
        disabled={importingRef.current}
      >
        {importingRef.current ? 'Importing...' : 'Import file'}
      </button>
    </div>
  );
}
