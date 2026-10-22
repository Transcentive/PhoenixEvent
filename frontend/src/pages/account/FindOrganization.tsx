import { useEffect, useId, useState, type FormEvent, type KeyboardEvent } from "react";
import { findOrganization, matchOrganizations, type OrganizationMatch } from "@/api/client";
import { Field, Heading, Shell, Text } from "./parts";

/*
  A1, at the general address (4A.7). The Organization field is a matching
  field: as the person types, it offers the organizations whose name or
  address name contains the letters typed, and choosing one goes to its login
  page (spec v1.02). The lookup below answers the same whatever it finds, and
  its list goes only by email.
*/
export default function FindOrganization() {
  const id = useId();
  const [typed, setTyped] = useState("");
  const [matches, setMatches] = useState<OrganizationMatch[]>([]);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [orgError, setOrgError] = useState<string>();
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState<string>();
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!typed.trim()) {
      setMatches([]);
      return;
    }
    const ask = new AbortController();
    const timer = setTimeout(() => {
      matchOrganizations(typed, ask.signal).then(
        (found) => {
          setMatches(found);
          setActive(-1);
        },
        () => undefined,
      );
    }, 150);
    return () => {
      clearTimeout(timer);
      ask.abort();
    };
  }, [typed]);

  /* The organization the buttons act on: the one whose name or address name
     is exactly what was typed, or else the only one offered. */
  function chosen(): OrganizationMatch | undefined {
    const t = typed.trim().toLowerCase();
    return matches.find((m) => m.name.toLowerCase() === t || m.slug === t) ?? (matches.length === 1 ? matches[0] : undefined);
  }

  function goTo(org: OrganizationMatch | undefined, page: "login" | "register") {
    if (!org) {
      setOrgError(matches.length ? "Choose your organization from the list." : "No organization matches what you typed.");
      setOpen(true);
      return;
    }
    window.location.assign(page === "login" ? org.login_url : org.login_url.replace(/login$/, "register"));
  }

  function go(e: FormEvent) {
    e.preventDefault();
    goTo(open && active >= 0 ? matches[active] : chosen(), "login");
  }

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown" && matches.length) {
      e.preventDefault();
      setOpen(true);
      setActive((a) => (a + 1) % matches.length);
    } else if (e.key === "ArrowUp" && matches.length) {
      e.preventDefault();
      setOpen(true);
      setActive((a) => (a <= 0 ? matches.length - 1 : a - 1));
    } else if (e.key === "Escape") {
      setOpen(false);
      setActive(-1);
    }
  }

  async function lookUp(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      setSent((await findOrganization(email)).message);
    } catch {
      setSent("If any organization has an account under this address, we've emailed you a link to each one.");
    } finally {
      setBusy(false);
    }
  }

  const listShown = open && matches.length > 0;
  return (
    <Shell>
      <div className="pe-login">
        <Heading
          title="Find your organization"
          org="Each organization has its own address for logging in. You work in one organization at a time."
        />
        <form className="pe-login__form" onSubmit={go} noValidate>
          <div className={orgError ? "pe-field pe-field--error" : "pe-field"}>
            <label className="pe-field__label" htmlFor={id}>
              Organization
            </label>
            <div className="pe-combo">
              <input
                className="pe-input"
                id={id}
                name="organization"
                autoComplete="off"
                role="combobox"
                aria-autocomplete="list"
                aria-expanded={listShown}
                aria-controls={`${id}-list`}
                aria-activedescendant={listShown && active >= 0 ? `${id}-${active}` : undefined}
                aria-invalid={orgError ? true : undefined}
                value={typed}
                onChange={(e) => {
                  setTyped(e.target.value);
                  setOpen(true);
                  setOrgError(undefined);
                }}
                onKeyDown={onKeyDown}
                onBlur={() => setOpen(false)}
              />
              {listShown && (
                <ul className="pe-combo__list" id={`${id}-list`} role="listbox" aria-label="Matching organizations">
                  {matches.map((m, i) => (
                    <li
                      key={m.slug}
                      id={`${id}-${i}`}
                      className="pe-combo__option"
                      role="option"
                      aria-selected={i === active}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        goTo(m, "login");
                      }}
                    >
                      <span className="pe-combo__name">{m.name}</span>
                      <span className="pe-combo__address">{m.slug}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
            {orgError && <span className="pe-field__error">{orgError}</span>}
            <span className="pe-field__hint">
              Each organization has its own login address, such as lincolnpta.{window.location.host}.
            </span>
          </div>
          <div className="pe-login__foot">
            <button className="pe-btn pe-btn--quiet pe-login__link" type="button" onClick={() => goTo(chosen(), "register")}>
              Create account
            </button>
            <button className="pe-btn pe-btn--primary" type="submit">
              Go to its login page
            </button>
          </div>
        </form>
        <form className="pe-login__section" onSubmit={lookUp} noValidate>
          <h2 className="pe-login__subtitle">Don't know your organization's address?</h2>
          {sent ? (
            <Text>{sent}</Text>
          ) : (
            <>
              <Field
                label="Email address"
                type="email"
                autoComplete="email"
                value={email}
                onChange={setEmail}
                hint="We'll email you a link to the login page of each organization where you have an account."
              />
              <div>
                <button className="pe-btn pe-btn--secondary" type="submit" disabled={busy}>
                  Email me the links
                </button>
              </div>
            </>
          )}
        </form>
      </div>
    </Shell>
  );
}
