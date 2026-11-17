import { useEffect, useState } from "react";

/**
 * Returns `value`, but only after it has stopped changing for `delayMs`.
 * Typing "elden" quickly makes one request for "elden" instead of five
 * (e, el, eld, elde, elden).
 */
export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(timer); // a new keystroke cancels the pending update
  }, [value, delayMs]);

  return debounced;
}
