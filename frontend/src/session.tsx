import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import * as api from "@/api/client";

/*
  The address and the session. The address says which organization this page
  belongs to (4A.5); the server decides it from the host, never the client.
  The access token is held in memory only. On load, and before it runs out,
  it is renewed from the sign-in cookie, which the page cannot read.
*/

interface SessionState {
  address: api.Address | null;
  addressError: boolean;
  session: api.Session | null;
  /* True until the first renewal has answered, so a reload does not flash
     the login page at someone who is logged in. */
  restoring: boolean;
  signIn: (s: api.Session) => void;
  signOut: () => Promise<void>;
}

const Context = createContext<SessionState | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [address, setAddress] = useState<api.Address | null>(null);
  const [addressError, setAddressError] = useState(false);
  const [session, setSession] = useState<api.Session | null>(null);
  const [restoring, setRestoring] = useState(true);

  useEffect(() => {
    api.getAddress().then(
      (a) => {
        setAddress(a);
        if (a.kind !== "organization" || a.logged_in_elsewhere) {
          setRestoring(false);
          return;
        }
        api
          .refresh()
          .then(setSession, () => setSession(null))
          .finally(() => setRestoring(false));
      },
      () => {
        setAddressError(true);
        setRestoring(false);
      },
    );
  }, []);

  /* Renew a minute before the access token runs out. */
  useEffect(() => {
    if (!session) return;
    const wait = Math.max(new Date(session.expires_at).getTime() - Date.now() - 60_000, 5_000);
    const timer = setTimeout(() => {
      api.refresh().then(setSession, () => setSession(null));
    }, wait);
    return () => clearTimeout(timer);
  }, [session]);

  const signIn = useCallback((s: api.Session) => setSession(s), []);
  const signOut = useCallback(async () => {
    try {
      await api.logout();
    } finally {
      setSession(null);
    }
  }, []);

  return (
    <Context.Provider value={{ address, addressError, session, restoring, signIn, signOut }}>
      {children}
    </Context.Provider>
  );
}

export function useSession(): SessionState {
  const value = useContext(Context);
  if (!value) throw new Error("useSession outside SessionProvider");
  return value;
}

/* The organization of this address. Only called on pages that the router
   shows at an organization's address. */
export function useOrganization(): api.Organization {
  const { address } = useSession();
  if (address?.kind !== "organization") throw new Error("not at an organization's address");
  return address.organization;
}
