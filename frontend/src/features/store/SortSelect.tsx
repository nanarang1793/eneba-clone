// Values must match the backend's sort options (catalog/params.go).
const SORT_LABELS: Record<string, string> = {
  relevance: "Relevance",
  popular: "Most popular",
  price_asc: "Price: low to high",
  price_desc: "Price: high to low",
  newest: "Newest",
};

export function SortSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <label className="flex items-center gap-2 text-sm text-muted">
      Sort by
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="h-9 rounded-md border border-line bg-surface px-3 text-sm text-white focus:border-accent focus:outline-none"
      >
        {Object.entries(SORT_LABELS).map(([v, label]) => (
          <option key={v} value={v}>
            {label}
          </option>
        ))}
      </select>
    </label>
  );
}
