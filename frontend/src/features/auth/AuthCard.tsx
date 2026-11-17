import type { InputHTMLAttributes, ReactNode } from "react";

/** The centred box both the sign-in and register pages use. */
export function AuthCard({ title, children, footer }: { title: string; children: ReactNode; footer: ReactNode }) {
  return (
    <div className="mx-auto mt-6 max-w-md">
      <div className="rounded-xl bg-surface p-8 shadow-xl">
        <h1 className="mb-6 text-2xl font-bold">{title}</h1>
        {children}
      </div>
      <p className="mt-4 text-center text-sm text-muted">{footer}</p>
    </div>
  );
}

/** A labelled input. The label is linked to the input so clicking it focuses the field. */
export function Field({ label, id, ...input }: { label: string; id: string } & InputHTMLAttributes<HTMLInputElement>) {
  return (
    <div>
      <label htmlFor={id} className="mb-1.5 block text-sm font-medium">
        {label}
      </label>
      <input
        id={id}
        {...input}
        className="h-11 w-full rounded-md border border-line bg-page px-3 text-sm placeholder:text-muted/60 focus:border-accent focus:outline-none"
      />
    </div>
  );
}

/** The server's message (e.g. "Wrong email or password"), announced to screen readers. */
export function FormError({ message }: { message: string | undefined }) {
  if (!message) return null;
  return (
    <p role="alert" className="rounded-md border border-discount/50 bg-discount/10 px-3 py-2 text-sm">
      {message}
    </p>
  );
}

export function SubmitButton({ pending, children }: { pending: boolean; children: ReactNode }) {
  return (
    <button
      type="submit"
      disabled={pending}
      className="h-11 w-full rounded-md bg-cta font-semibold text-page hover:bg-cta-hover disabled:opacity-60"
    >
      {pending ? "Please wait…" : children}
    </button>
  );
}
