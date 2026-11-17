import { useEffect, useState } from "react";

/**
 * Seconds left until `deadline` (milliseconds on this browser's clock),
 * updated every second. Never below 0.
 *
 * The deadline is built from the server's "seconds_left" plus the moment we
 * received it, so the countdown only ever uses ONE clock (this browser's).
 * Comparing the server's expires_at with the browser's clock broke when the
 * two clocks disagreed: a Docker VM 14 hours behind made every new order
 * look expired immediately.
 */
export function useCountdown(deadline: number | undefined): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);
  if (deadline === undefined) return 0;
  return Math.max(0, Math.floor((deadline - now) / 1000));
}
