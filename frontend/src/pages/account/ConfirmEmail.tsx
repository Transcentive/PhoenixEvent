import { useEffect, useRef, useState } from "react";
import { Navigate, useSearchParams } from "react-router-dom";
import { confirmEmail } from "@/api/client";
import LinkEnded from "./LinkEnded";
import { Shell, Text } from "./parts";

/* Where the confirmation link in the email lands. It confirms, then shows
   the login page; a used or expired link shows E2. */
export default function ConfirmEmail() {
  const token = useSearchParams()[0].get("token") ?? "";
  const [state, setState] = useState<"working" | "done" | "ended">("working");
  const asked = useRef(false);

  useEffect(() => {
    if (asked.current) return;
    asked.current = true;
    confirmEmail(token).then(
      () => setState("done"),
      () => setState("ended"),
    );
  }, [token]);

  if (state === "ended") return <LinkEnded />;
  if (state === "done") {
    return <Navigate to="/login" replace state={{ notice: "Your email address is confirmed. You can log in." }} />;
  }
  return (
    <Shell>
      <div className="pe-login">
        <Text>Confirming your email address.</Text>
      </div>
    </Shell>
  );
}
