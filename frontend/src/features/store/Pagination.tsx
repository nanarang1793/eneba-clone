type Props = { page: number; totalPages: number; onChange: (page: number) => void };

/** Prev · 1 … 4 5 6 … 9 · Next */
export function Pagination({ page, totalPages, onChange }: Props) {
  if (totalPages <= 1) return null;

  const btn = "h-9 min-w-9 rounded-md px-3 text-sm font-medium transition disabled:opacity-40";

  return (
    <nav aria-label="Pagination" className="mt-8 flex flex-wrap items-center justify-center gap-1.5">
      <button className={`${btn} bg-surface hover:bg-surface-hover`} disabled={page <= 1} onClick={() => onChange(page - 1)}>
        ‹ Prev
      </button>

      {pageList(page, totalPages).map((p, i) =>
        p === "…" ? (
          <span key={`gap${i}`} className="px-1 text-muted">
            …
          </span>
        ) : (
          <button
            key={p}
            onClick={() => onChange(p)}
            aria-current={p === page ? "page" : undefined}
            className={`${btn} ${p === page ? "bg-cta text-page" : "bg-surface hover:bg-surface-hover"}`}
          >
            {p}
          </button>
        ),
      )}

      <button
        className={`${btn} bg-surface hover:bg-surface-hover`}
        disabled={page >= totalPages}
        onClick={() => onChange(page + 1)}
      >
        Next ›
      </button>
    </nav>
  );
}

/**
 * Which page numbers to show: always the first and last, plus one page on each
 * side of the current one, with "…" for the gaps.
 * pageList(5, 9) -> [1, "…", 4, 5, 6, "…", 9]
 */
export function pageList(page: number, total: number): (number | "…")[] {
  const out: (number | "…")[] = [];
  for (let p = 1; p <= total; p++) {
    if (p === 1 || p === total || Math.abs(p - page) <= 1) out.push(p);
    else if (out[out.length - 1] !== "…") out.push("…");
  }
  return out;
}
