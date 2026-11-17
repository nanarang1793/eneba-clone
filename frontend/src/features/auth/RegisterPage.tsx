import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate, useSearchParams } from "react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { register } from "../../api/auth";
import { AuthCard, Field, FormError, SubmitButton } from "./AuthCard";
import { ME_KEY, safeNext, useCurrentUser } from "./useCurrentUser";

/**
 * /register: creating an account also signs you in (the backend sets the
 * session cookie in the same response).
 * The browser checks the basics (required, email format, 8+ characters);
 * the backend checks everything again, because browser checks are easy to skip.
 */
export function RegisterPage() {
  const [searchParams] = useSearchParams();
  const next = safeNext(searchParams.get("next"));
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user } = useCurrentUser();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const mutation = useMutation({
    mutationFn: () => register(email, password, name),
    onSuccess: (u) => {
      queryClient.setQueryData(ME_KEY, u);
      navigate(next, { replace: true });
    },
  });

  if (user && !mutation.isPending) return <Navigate to={next} replace />;

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    mutation.mutate();
  }

  return (
    <AuthCard
      title="Create an account"
      footer={
        <>
          Already have an account?{" "}
          <Link to={`/login?next=${encodeURIComponent(next)}`} className="text-accent hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4">
        <Field label="Name (optional)" id="name" autoComplete="nickname" maxLength={40} value={name}
          onChange={(e) => setName(e.target.value)} />
        <Field label="Email" id="email" type="email" autoComplete="email" required value={email}
          onChange={(e) => setEmail(e.target.value)} />
        <Field label="Password" id="password" type="password" autoComplete="new-password" required minLength={8}
          maxLength={72} placeholder="At least 8 characters" value={password}
          onChange={(e) => setPassword(e.target.value)} />
        <FormError message={mutation.error?.message} />
        <SubmitButton pending={mutation.isPending}>Create account</SubmitButton>
      </form>
    </AuthCard>
  );
}
