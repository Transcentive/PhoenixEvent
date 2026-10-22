import { useNavigate } from "react-router-dom";
import { useSession } from "@/session";
import { Heading, Note, Shell, Text } from "./parts";

/* Stands in for the main page until the later phases build it: it shows
   whom the session belongs to, and logs out. */
export default function Home() {
  const { address, session, signOut } = useSession();
  const navigate = useNavigate();
  if (address?.kind !== "organization" || !session) return null;
  const { account } = session;

  async function logOut() {
    await signOut().catch(() => undefined);
    navigate("/logged-out", { replace: true });
  }

  return (
    <Shell
      right={
        <button className="pe-btn pe-btn--quiet pe-topbar__end" type="button" onClick={logOut}>
          Log out
        </button>
      }
    >
      <div className="pe-login">
        <Heading title="You are logged in" org={address.organization.name} />
        <Text>{account.username ?? account.email}</Text>
        <Note>The main page comes with the later phases.</Note>
      </div>
    </Shell>
  );
}
