import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { ApiError, linkPurpose, resetPassword } from "@/api/client";
import LinkEnded from "./LinkEnded";
import { Alert, Heading, PASSWORD_HINT, PasswordField, Shell, Text } from "./parts";

/* A6. It serves the reset link (A5) and the link a new organization's first
   administrator gets (4A.6). The link is checked first, so an ended one
   shows E2 before anyone types a password. */
export default function ResetPassword() {
  const token = useSearchParams()[0].get("token") ?? "";
  const navigate = useNavigate();
  const [state, setState] = useState<"checking" | "ready" | "ended">("checking");
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [fieldError, setFieldError] = useState<string>();
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    linkPurpose(token).then(
      (link) => {
        setLogin(link.login);
        setState("ready");
      },
      () => setState("ended"),
    );
  }, [token]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(undefined);
    setFieldError(undefined);
    try {
      await resetPassword(token, password);
      navigate("/login", { replace: true, state: { notice: "Your password is saved. Log in with it." } });
    } catch (err) {
      if (err instanceof ApiError && err.code === "link_ended") setState("ended");
      else if (err instanceof ApiError && err.fields.length > 0) setFieldError(err.fields[0]?.message);
      else setError(err instanceof ApiError ? err.message : "Something went wrong on our side. Try again.");
    } finally {
      setBusy(false);
    }
  }

  if (state === "ended") return <LinkEnded />;
  return (
    <Shell>
      {state === "checking" ? (
        <div className="pe-login">
          <Text>Checking the link.</Text>
        </div>
      ) : (
        <form className="pe-login" onSubmit={submit} noValidate>
          <Heading title="Set a new password" org={`For ${login}`} />
          {/* Lets a password manager save the new password under this account. */}
          <input type="text" name="username" autoComplete="username" value={login} readOnly hidden />
          {error && <Alert>{error}</Alert>}
          <PasswordField
            label="New password"
            autoComplete="new-password"
            value={password}
            onChange={setPassword}
            error={fieldError}
            hint={PASSWORD_HINT}
          />
          <div className="pe-login__foot">
            <Link className="pe-link" to="/login">
              Back to log in
            </Link>
            <button className="pe-btn pe-btn--primary" type="submit" disabled={busy}>
              Save password
            </button>
          </div>
        </form>
      )}
    </Shell>
  );
}
