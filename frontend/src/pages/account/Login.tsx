import { useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { ApiError, login } from "@/api/client";
import { useSession } from "@/session";
import { Alert, Field, Heading, Notice, PasswordField, Shell } from "./parts";

/* A2, with E1 for a failed attempt. */
export default function Login() {
  const { address, signIn } = useSession();
  const navigate = useNavigate();
  const notice = (useLocation().state as { notice?: string } | null)?.notice;
  const [who, setWho] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);
  if (address?.kind !== "organization") return null;
  const elsewhere = address.logged_in_elsewhere;

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    try {
      signIn(await login(who, password));
      navigate("/", { replace: true });
    } catch (err) {
      // E1: the password field clears after a failed attempt.
      setPassword("");
      if (err instanceof ApiError && err.code === "logged_in_elsewhere") {
        window.location.reload();
        return;
      }
      setError(err instanceof ApiError ? err.message : "Something went wrong on our side. Try again.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Shell>
      <div className="pe-login">
        <Heading
          title="Log in"
          org={
            <>
              {address.organization.name} · <a className="pe-link" href={address.general_url}>Change organization</a>
            </>
          }
        />
        {elsewhere ? (
          <>
            <Alert>You are logged in to {elsewhere.name}.</Alert>
            <div className="pe-login__foot">
              <span />
              <a className="pe-btn pe-btn--primary" href={elsewhere.logout_url}>
                Log out
              </a>
            </div>
          </>
        ) : (
          <form className="pe-login__form" onSubmit={submit} noValidate>
            {notice && !error && <Notice>{notice}</Notice>}
            {error && <Alert>{error}</Alert>}
            <Field
              label="Username"
              name="username"
              autoComplete="username"
              value={who}
              onChange={setWho}
              hint="Your username or your email address. Both work, even after you choose a username."
            />
            <PasswordField label="Password" autoComplete="current-password" value={password} onChange={setPassword} />
            <div className="pe-login__foot">
              <Link className="pe-link" to="/forgot-password">
                Forgot password?
              </Link>
              <button className="pe-btn pe-btn--primary" type="submit" disabled={busy}>
                Log in
              </button>
            </div>
          </form>
        )}
      </div>
      {!elsewhere && (
        <p className="pe-note">
          New to PhoenixEvent?{" "}
          <Link className="pe-link" to="/register">
            Create account
          </Link>
        </p>
      )}
    </Shell>
  );
}
