const euro = new Intl.NumberFormat("en-IE", { style: "currency", currency: "EUR" });

/** 2119 -> "€21.19". Prices travel as integer cents and are only formatted here. */
export function formatPrice(cents: number) {
  return euro.format(cents / 100);
}

export function Price({ cents, className = "" }: { cents: number; className?: string }) {
  return <span className={className}>{formatPrice(cents)}</span>;
}
