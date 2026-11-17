import { useEffect } from "react";

/** Sets the browser tab title: "Cart | Eneba clone". null gives just "Eneba clone". */
export function usePageTitle(title: string | null | undefined) {
  useEffect(() => {
    document.title = title ? `${title} | Eneba clone` : "Eneba clone";
  }, [title]);
}
