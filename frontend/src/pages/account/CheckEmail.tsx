import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { resendConfirmation } from "@/api/client";
import { Badge, Heading, Note, Notice, Shell, Text } from "./parts";

/* A4. */
export default function CheckEmail() {
  const email = (useLocation().state as { email?: string } | null)?.email;
  const [resent, setResent] = useState(false);

  async function resend() {
    if (!email) return;
    await resendConfirmation(email).catch(() => undefined);
    setResent(true);
  }

  return (
    <Shell>
      <div className="pe-login">
        <Heading title="Check your email" />
        <Badge role="caution">Waiting for confirmation</Badge>
        <Text>
          We sent a confirmation link to {email ?? "your email address"}. Open the link to confirm your account.
        </Text>
        <Note>The link works once and expires after 24 hours.</Note>
        <Note>You can log in once your email address is confirmed.</Note>
        {resent && <Notice>We sent the email again.</Notice>}
        <div className="pe-login__foot">
          <Link className="pe-link" to="/login">
            Back to log in
          </Link>
          {email && (
            <button className="pe-btn pe-btn--secondary" type="button" onClick={resend}>
              Send the email again
            </button>
          )}
        </div>
      </div>
    </Shell>
  );
}
