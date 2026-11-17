import type { OrderStatus } from "../../api/types";

const STYLES: Record<OrderStatus, { label: string; className: string }> = {
  paid: { label: "Paid", className: "bg-region/20 text-region" },
  pending: { label: "Awaiting payment", className: "bg-cta/20 text-cta" },
  failed: { label: "Payment failed", className: "bg-discount/20 text-discount" },
  expired: { label: "Expired", className: "bg-line text-muted" },
};

export function StatusBadge({ status }: { status: OrderStatus }) {
  const s = STYLES[status];
  return <span className={`rounded px-2 py-0.5 text-xs font-semibold ${s.className}`}>{s.label}</span>;
}
