import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { forgotPassword } from "@/api/client";
import { Field, Heading, Note, Shell, Text } from "./parts";

/* A5. It answers the same whether or not an account matches. */
export default function ForgotPassword() {
  const [who, setWho] = useState("");
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    await forgotPassword(who).catch(() => undefined);
    setBusy(false);
    setSent(true);
  }

  return (
    <Shell>
      {sent ? (
        <div className="pe-login">
          <Heading title="Check your email" />
          <Text>If an account matches what you entered, we sent a link to set a new password.</Text>
          <Note>The link works once and expires after 1 hour.</Note>
          <div className="pe-login__foot">
            <Link className="pe-link" to="/login">
              Back to log in
            </Link>
          </div>
        </div>
      ) : (
        <form className="pe-login" onSubmit={submit} noValidate>
          <Heading title="Forgot password?" />
          <Text>Enter your username or email address. We send a link to set a new password.</Text>
          <Field label="Username or email address" name="username" autoComplete="username" value={who} onChange={setWho} />
          <div className="pe-login__foot">
            <Link className="pe-link" to="/login">
              Back to log in
            </Link>
            <button className="pe-btn pe-btn--primary" type="submit" disabled={busy}>
              Send reset link
            </button>
          </div>
        </form>
      )}
    </Shell>
  );
}
