import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate, useSearchParams } from "react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { login } from "../../api/auth";
import { AuthCard, Field, FormError, SubmitButton } from "./AuthCard";
import { ME_KEY, safeNext, useCurrentUser } from "./useCurrentUser";
import { usePageTitle } from "../../components/usePageTitle";

/**
 * /login?next=/account
 * After signing in, the user goes back to the page that sent them here.
 */
export function LoginPage() {
  const [searchParams] = useSearchParams();
  const next = safeNext(searchParams.get("next"));
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user } = useCurrentUser();
  usePageTitle("Sign in");

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const mutation = useMutation({
    mutationFn: () => login(email, password),
    onSuccess: (u) => {
      queryClient.setQueryData(ME_KEY, u); // header updates immediately
      navigate(next, { replace: true }); // "back" shouldn't return to the login form
    },
  });

  // Already signed in (e.g. opened /login in a new tab): skip the form.
  if (user && !mutation.isPending) return <Navigate to={next} replace />;

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    mutation.mutate();
  }

  return (
    <AuthCard
      title="Sign in"
      footer={
        <>
          New here?{" "}
          <Link to={`/register?next=${encodeURIComponent(next)}`} className="text-accent hover:underline">
            Create an account
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4">
        <Field label="Email" id="email" type="email" autoComplete="email" required value={email}
          onChange={(e) => setEmail(e.target.value)} />
        <Field label="Password" id="password" type="password" autoComplete="current-password" required value={password}
          onChange={(e) => setPassword(e.target.value)} />
        <FormError message={mutation.error?.message} />
        <SubmitButton pending={mutation.isPending}>Sign in</SubmitButton>
      </form>
      <p className="mt-4 text-xs text-muted">
        Demo account: <span className="text-white">demo@eneba.test</span> / <span className="text-white">demo1234</span>
      </p>
    </AuthCard>
  );
}
