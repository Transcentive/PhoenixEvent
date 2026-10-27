import type { ReactNode } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { getProfile } from "@/api/client";
import { useSession } from "@/session";

/*
  The frame of every page for someone logged in, as the wireframes draw it:
  the organization, and the name shown (8B.2), which links to the profile
  (8B.3), with Log out.
*/
export function useOwnProfile() {
  const { session } = useSession();
  return useQuery({
    queryKey: ["profile", session?.account.id],
    queryFn: () => getProfile(session!.access_token),
    enabled: !!session,
  });
}

const BUILDING = (
  <svg className="pe-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <rect x="5" y="4" width="14" height="16" rx="1" />
    <path d="M9 8h2M13 8h2M9 12h2M13 12h2M10 20v-4h4v4" />
  </svg>
);

export default function SignedInShell({ children }: { children: ReactNode }) {
  const { address, session, signOut } = useSession();
  const navigate = useNavigate();
  const profile = useOwnProfile();
  if (address?.kind !== "organization" || !session) return null;

  async function logOut() {
    await signOut().catch(() => undefined);
    navigate("/logged-out", { replace: true });
  }

  return (
    <div className="pe-app">
      <header className="pe-topbar">
        <span className="pe-topbar__name">PhoenixEvent</span>
        <span className="pe-topbar__org">
          {BUILDING}
          {address.organization.name}
        </span>
        <span className="pe-topbar__end" />
        <Link className="pe-link pe-topbar__person" to="/profile">
          {profile.data?.name ?? session.account.username ?? session.account.email}
        </Link>
        <button className="pe-btn pe-btn--secondary" type="button" onClick={logOut}>
          Log out
        </button>
      </header>
      <main className="pe-page-wide">{children}</main>
    </div>
  );
}
