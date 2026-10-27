import { useSession } from "@/session";
import SignedInShell from "@/pages/SignedInShell";
import { Note } from "./parts";

/* Stands in for the main page until the later phases build it. */
export default function Home() {
  const { address } = useSession();
  if (address?.kind !== "organization") return null;
  return (
    <SignedInShell>
      <h1 className="pe-page-title">{address.organization.name}</h1>
      <Note>The main page comes with the later phases. Your name above opens your profile.</Note>
    </SignedInShell>
  );
}
