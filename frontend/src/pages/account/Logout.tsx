import { useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useSession } from "@/session";
import { Shell, Text } from "./parts";

/*
  /logout logs this browser out of this organization (4A.5, rule 3). Another
  organization's login page sends the browser here with ?return= its own
  login address, and the browser goes back there afterwards. Only an address
  under the same domain is followed.
*/
export function safeReturn(raw: string | null, hostname: string): string | null {
  if (!raw) return null;
  try {
    const target = new URL(raw);
    const domain = hostname.split(".").slice(1).join(".");
    const ok = target.hostname === domain || target.hostname.endsWith(`.${domain}`);
    return ok && /^https?:$/.test(target.protocol) ? target.href : null;
  } catch {
    return null;
  }
}

export default function Logout() {
  const { signOut } = useSession();
  const navigate = useNavigate();
  const back = safeReturn(useSearchParams()[0].get("return"), window.location.hostname);
  const done = useRef(false);

  useEffect(() => {
    if (done.current) return;
    done.current = true;
    signOut()
      .catch(() => undefined)
      .then(() => {
        if (back) window.location.assign(back);
        else navigate("/logged-out", { replace: true });
      });
  }, [back, navigate, signOut]);

  return (
    <Shell>
      <div className="pe-login">
        <Text>Logging out.</Text>
      </div>
    </Shell>
  );
}
