import { useEffect, useState, type FormEvent } from "react";
import type { FacetCount, Filters } from "../../api/types";
import { formatPrice } from "../../components/ui/Price";
import type { ListKey } from "./useStoreParams";

type Props = {
  meta: Filters | undefined; // from GET /api/filters (all options + price bounds)
  facets: { platforms: FacetCount[]; regions: FacetCount[]; types: FacetCount[] } | undefined; // counts for the current search
  selected: { platforms: string[]; regions: string[]; types: string[]; minPrice: string; maxPrice: string };
  onToggle: (key: ListKey, slug: string) => void;
  onPrice: (min: string, max: string) => void;
  onClear: () => void;
};

export function FilterSidebar({ meta, facets, selected, onToggle, onPrice, onClear }: Props) {
  const hasFilters =
    selected.platforms.length > 0 ||
    selected.regions.length > 0 ||
    selected.types.length > 0 ||
    selected.minPrice !== "" ||
    selected.maxPrice !== "";

  return (
    <aside className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-bold">Filters</h2>
        {hasFilters && (
          <button onClick={onClear} className="text-sm text-accent hover:underline">
            Clear all
          </button>
        )}
      </div>

      <PriceFilter
        minPrice={selected.minPrice}
        maxPrice={selected.maxPrice}
        bounds={meta?.price}
        onApply={onPrice}
      />

      <CheckboxGroup
        title="Type"
        options={facets?.types ?? meta?.types.map((t) => ({ ...t, count: -1 })) ?? []}
        selected={selected.types}
        onToggle={(slug) => onToggle("type", slug)}
      />
      <CheckboxGroup
        title="Platform"
        options={facets?.platforms ?? meta?.platforms.map((p) => ({ ...p, count: -1 })) ?? []}
        selected={selected.platforms}
        onToggle={(slug) => onToggle("platform", slug)}
      />
      <CheckboxGroup
        title="Region"
        options={facets?.regions ?? meta?.regions.map((r) => ({ ...r, count: -1 })) ?? []}
        selected={selected.regions}
        onToggle={(slug) => onToggle("region", slug)}
      />
    </aside>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg bg-surface p-4">
      <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted">{title}</h3>
      {children}
    </section>
  );
}

/**
 * Checkbox list with counts. An option with 0 matches is disabled, unless it's
 * already ticked (so it can still be unticked). count -1 means "not loaded yet".
 */
function CheckboxGroup(props: {
  title: string;
  options: FacetCount[];
  selected: string[];
  onToggle: (slug: string) => void;
}) {
  return (
    <Section title={props.title}>
      <ul className="space-y-1">
        {props.options.map((o) => {
          const checked = props.selected.includes(o.slug);
          const disabled = o.count === 0 && !checked;
          return (
            <li key={o.slug}>
              <label
                className={`flex cursor-pointer items-center gap-2.5 rounded px-1 py-1 text-sm hover:bg-surface-hover ${
                  disabled ? "cursor-not-allowed opacity-40" : ""
                }`}
              >
                <input
                  type="checkbox"
                  checked={checked}
                  disabled={disabled}
                  onChange={() => props.onToggle(o.slug)}
                  className="size-4 accent-cta"
                />
                <span className="flex-1">{o.name}</span>
                {o.count >= 0 && <span className="text-xs text-muted">{o.count}</span>}
              </label>
            </li>
          );
        })}
      </ul>
    </Section>
  );
}

/**
 * Min/max inputs. They keep their own text while you type and only write to
 * the URL when you press Apply (otherwise every keystroke would refetch).
 */
function PriceFilter(props: {
  minPrice: string;
  maxPrice: string;
  bounds: { min_cents: number; max_cents: number } | undefined;
  onApply: (min: string, max: string) => void;
}) {
  const [min, setMin] = useState(props.minPrice);
  const [max, setMax] = useState(props.maxPrice);

  // If the URL changes from outside (Clear all, back button), show the new values.
  useEffect(() => setMin(props.minPrice), [props.minPrice]);
  useEffect(() => setMax(props.maxPrice), [props.maxPrice]);

  const invalid = min !== "" && max !== "" && Number(min) > Number(max);

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!invalid) props.onApply(min, max);
  }

  const inputClass =
    "h-9 w-full rounded-md border border-line bg-page px-2 text-sm placeholder:text-muted/60 focus:border-accent focus:outline-none";

  return (
    <Section title="Price (€)">
      <form onSubmit={onSubmit} className="space-y-2">
        <div className="flex items-center gap-2">
          <input
            type="number"
            min={0}
            step="0.01"
            inputMode="decimal"
            value={min}
            onChange={(e) => setMin(e.target.value)}
            placeholder={props.bounds ? formatPrice(props.bounds.min_cents) : "Min"}
            aria-label="Minimum price in euros"
            className={inputClass}
          />
          <span className="text-muted">–</span>
          <input
            type="number"
            min={0}
            step="0.01"
            inputMode="decimal"
            value={max}
            onChange={(e) => setMax(e.target.value)}
            placeholder={props.bounds ? formatPrice(props.bounds.max_cents) : "Max"}
            aria-label="Maximum price in euros"
            className={inputClass}
          />
        </div>
        {invalid && <p className="text-xs text-discount">Min can't be higher than max.</p>}
        <button
          type="submit"
          disabled={invalid}
          className="h-9 w-full rounded-md bg-cta text-sm font-semibold text-page hover:bg-cta-hover disabled:opacity-50"
        >
          Apply
        </button>
      </form>
    </Section>
  );
}
