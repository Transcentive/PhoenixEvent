import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ApiError, register } from "@/api/client";
import { useOrganization } from "@/session";
import { Alert, Field, Heading, PASSWORD_HINT, PasswordField, Shell } from "./parts";

/* A3, with E4 for a field that is refused. */
export default function CreateAccount() {
  const org = useOrganization();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    try {
      const made = await register(email, username, password);
      navigate("/check-email", { state: { email: made.email } });
    } catch (err) {
      if (err instanceof ApiError && err.fields.length > 0) {
        setFields(Object.fromEntries(err.fields.map((f) => [f.field, f.message])));
      } else {
        setFields({});
        setError(err instanceof ApiError ? err.message : "Something went wrong on our side. Try again.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <Shell>
      <div className="pe-login">
        <Heading title="Create account" org={`${org.name} · An account here works at this organization only.`} />
        <form className="pe-login__form" onSubmit={submit} noValidate>
          {error && <Alert>{error}</Alert>}
          <Field
            label="Email address"
            type="email"
            autoComplete="email"
            value={email}
            onChange={setEmail}
            error={fields.email}
            hint="We send a confirmation link to this address."
          />
          <Field
            label="Username (optional)"
            autoComplete="username"
            value={username}
            onChange={setUsername}
            error={fields.username}
            hint="Leave this empty to use your email address as your username."
          />
          <PasswordField
            label="Password"
            autoComplete="new-password"
            value={password}
            onChange={setPassword}
            error={fields.password}
            hint={PASSWORD_HINT}
          />
          <div className="pe-login__foot">
            <Link className="pe-link" to="/login">
              Back to log in
            </Link>
            <button className="pe-btn pe-btn--primary" type="submit" disabled={busy}>
              Create account
            </button>
          </div>
        </form>
      </div>
    </Shell>
  );
}
