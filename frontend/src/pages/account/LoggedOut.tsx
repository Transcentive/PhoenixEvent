import { Link } from "react-router-dom";
import { useSession } from "@/session";
import { Heading, Note, Shell, Text } from "./parts";

/* A7. */
export default function LoggedOut() {
  const { address } = useSession();
  if (address?.kind !== "organization") return null;
  return (
    <Shell>
      <div className="pe-login">
        <Heading title="You are logged out" />
        <Text>Your session with {address.organization.name} has ended.</Text>
        <Note>To work in another organization, log in to that organization.</Note>
        <div className="pe-login__foot">
          <a className="pe-link" href={address.general_url}>
            Log in to another organization
          </a>
          <Link className="pe-btn pe-btn--primary" to="/login">
            Log in again
          </Link>
        </div>
      </div>
    </Shell>
  );
}
