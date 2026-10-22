import { Link } from "react-router-dom";
import { Badge, Heading, Shell, Text } from "./parts";

/* E2: one screen for both kinds of link. */
export default function LinkEnded() {
  return (
    <Shell>
      <div className="pe-login">
        <Heading title="This link has ended" />
        <Badge role="critical">Expired or already used</Badge>
        <Text>
          Each link works once. A confirmation link expires after 24 hours, and a password reset link after 1 hour. Send
          a new one to continue.
        </Text>
        <div className="pe-login__foot">
          <Link className="pe-link" to="/login">
            Back to log in
          </Link>
          <Link className="pe-btn pe-btn--primary" to="/forgot-password">
            Send a new link
          </Link>
        </div>
      </div>
    </Shell>
  );
}
