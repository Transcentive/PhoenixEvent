import { useId, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import {
  ApiError,
  changeEmail,
  updateProfile,
  updateVisibility,
  type Level,
  type NameShown,
  type Profile,
} from "@/api/client";
import { useSession } from "@/session";
import SignedInShell, { useOwnProfile } from "@/pages/SignedInShell";
import { Alert, Field, Notice, PasswordField } from "@/pages/account/parts";

/*
  A8, Profile, private view (spec 8B.3, 8B.4). What the person enters and
  the visibility that acts on it are separate sections, each with its own
  Save (v1.06).
*/

const LEVELS: [Level, string][] = [
  ["public", "Public"],
  ["organization", "Organization members"],
  ["event", "Event participants"],
  ["station", "Station members"],
  ["private", "Private"],
];

function failure(err: unknown): { fields: Record<string, string>; message?: string } {
  if (err instanceof ApiError && err.fields.length > 0) {
    return { fields: Object.fromEntries(err.fields.map((f) => [f.field, f.message])) };
  }
  return { fields: {}, message: err instanceof ApiError ? err.message : "Something went wrong on our side. Try again." };
}

function useStore() {
  const { session } = useSession();
  const queryClient = useQueryClient();
  return (p: Profile) => queryClient.setQueryData(["profile", session?.account.id], p);
}

function ProfileSection({ profile, onChange }: { profile: Profile; onChange: () => void }) {
  const { session } = useSession();
  const store = useStore();
  const [username, setUsername] = useState(profile.username ?? "");
  const [first, setFirst] = useState(profile.first_name ?? "");
  const [middle, setMiddle] = useState(profile.middle_name ?? "");
  const [last, setLast] = useState(profile.last_name ?? "");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState<string>();
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);

  async function save(e: FormEvent) {
    e.preventDefault();
    if (!session) return;
    setBusy(true);
    setSaved(false);
    try {
      store(await updateProfile(session.access_token, { username, first_name: first, middle_name: middle, last_name: last }));
      setFields({});
      setError(undefined);
      setSaved(true);
    } catch (err) {
      const f = failure(err);
      setFields(f.fields);
      setError(f.message ?? f.fields.profile);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="pe-card pe-stack" onSubmit={save} noValidate aria-labelledby="profile-heading">
      <h2 className="pe-card__title" id="profile-heading">
        Profile
      </h2>
      {error && <Alert>{error}</Alert>}
      {saved && <Notice>Your profile is saved.</Notice>}
      <div className="pe-readonly pe-readonly--action">
        <div>
          <span className="pe-field__label">Email address</span>
          <span className="pe-readonly__value">{profile.email}</span>
        </div>
        <button className="pe-btn pe-btn--secondary" type="button" onClick={onChange}>
          Change
        </button>
      </div>
      <Field label="Username (optional)" autoComplete="username" value={username} onChange={setUsername} error={fields.username} />
      <div className="pe-names">
        <Field label="First name" autoComplete="given-name" value={first} onChange={setFirst} error={fields.first_name} />
        <Field label="Middle name (optional)" autoComplete="additional-name" value={middle} onChange={setMiddle} />
        <Field label="Last name" autoComplete="family-name" value={last} onChange={setLast} error={fields.last_name} />
      </div>
      <div className="pe-readonly">
        <span className="pe-field__label">Roles</span>
        <span className="pe-readonly__value">{profile.roles.length ? profile.roles.join(", ") : "None"}</span>
        <span className="pe-field__hint">Roles cannot be changed here.</span>
      </div>
      <div className="pe-actions">
        <button className="pe-btn pe-btn--primary" type="submit" disabled={busy}>
          Save profile
        </button>
      </div>
    </form>
  );
}

function LevelField({
  label,
  value,
  onChange,
  orgName,
  error,
}: {
  label: string;
  value: Level;
  onChange: (l: Level) => void;
  orgName: string;
  error?: string;
}) {
  const id = useId();
  return (
    <>
      <div className={error ? "pe-field pe-field--error" : "pe-field"}>
        <label className="pe-field__label" htmlFor={id}>
          {label}
        </label>
        <select className="pe-input" id={id} value={value} onChange={(e) => onChange(e.target.value as Level)}>
          {LEVELS.map(([v, text]) => (
            <option key={v} value={v}>
              {text}
            </option>
          ))}
        </select>
        {error && <span className="pe-field__error">{error}</span>}
      </div>
      {value === "public" && (
        <p className="pe-warning" role="note">
          Public means anyone can see this without logging in, including people outside {orgName}. Choose Public only for
          details you are happy to share with anyone.
        </p>
      )}
    </>
  );
}

function VisibilitySection({ profile, orgName }: { profile: Profile; orgName: string }) {
  const { session } = useSession();
  const store = useStore();
  const [emailLevel, setEmailLevel] = useState(profile.email_level);
  const [usernameLevel, setUsernameLevel] = useState(profile.username_level);
  const [nameLevel, setNameLevel] = useState(profile.name_level);
  const [rolesLevel, setRolesLevel] = useState(profile.roles_level);
  /* Untouched, the choice stays with the rule of 8B.2. */
  const [shown, setShown] = useState<NameShown>(profile.name_shown);
  const [shownTouched, setShownTouched] = useState(false);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState<string>();
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);

  const personal =
    profile.first_name && profile.last_name
      ? [profile.first_name, profile.middle_name, profile.last_name].filter(Boolean).join(" ")
      : "";
  /* Only the names already saved in the profile can be chosen. */
  const choices: [NameShown, string, string][] = [
    ["email", "Email address", profile.email],
    ["username", "Username", profile.username ?? ""],
    ["personal_name", "Personal name", personal],
  ];

  async function save(e: FormEvent) {
    e.preventDefault();
    if (!session) return;
    setBusy(true);
    setSaved(false);
    try {
      const p = await updateVisibility(session.access_token, {
        name_shown: shownTouched ? shown : (profile.name_choice ?? ""),
        email_level: emailLevel,
        username_level: usernameLevel,
        name_level: nameLevel,
        roles_level: rolesLevel,
      });
      store(p);
      setShown(p.name_shown);
      setShownTouched(false);
      setFields({});
      setError(undefined);
      setSaved(true);
    } catch (err) {
      const f = failure(err);
      setFields(f.fields);
      setError(f.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="pe-card pe-stack" onSubmit={save} noValidate aria-labelledby="visibility-heading">
      <h2 className="pe-card__title" id="visibility-heading">
        Profile visibility
      </h2>
      {error && <Alert>{error}</Alert>}
      {saved && <Notice>Your profile visibility is saved.</Notice>}
      <LevelField label="Email address" value={emailLevel} onChange={setEmailLevel} orgName={orgName} error={fields.email_level} />
      <LevelField label="Username" value={usernameLevel} onChange={setUsernameLevel} orgName={orgName} error={fields.username_level} />
      <LevelField label="Personal name" value={nameLevel} onChange={setNameLevel} orgName={orgName} error={fields.name_level} />
      <LevelField label="Roles" value={rolesLevel} onChange={setRolesLevel} orgName={orgName} error={fields.roles_level} />
      <fieldset className="pe-choices">
        <legend className="pe-field__label">Name shown</legend>
        {choices.map(([value, label, text]) => (
          <label key={value} className="pe-choice">
            <input
              type="radio"
              name="name-shown"
              value={value}
              checked={shown === value}
              disabled={!text}
              onChange={() => {
                setShown(value);
                setShownTouched(true);
              }}
            />
            {label}: {text || "not saved"}
          </label>
        ))}
        {fields.name_shown && <span className="pe-field__error">{fields.name_shown}</span>}
      </fieldset>
      <p className="pe-note">
        Shown in the page header and on Members and roles. Only what you have saved in your profile can be chosen. One of
        email address, username or personal name must not be Private, and the name shown must be visible at least to
        organization members. Private stays private, on Members and roles too.
      </p>
      <div className="pe-actions">
        <button className="pe-btn pe-btn--primary" type="submit" disabled={busy}>
          Save
        </button>
      </div>
    </form>
  );
}

function ChangeEmailSection({ onDone }: { onDone: (changed: boolean) => void }) {
  const { session } = useSession();
  const store = useStore();
  const [newEmail, setNewEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!session) return;
    try {
      store(await changeEmail(session.access_token, newEmail, password));
      onDone(true);
    } catch (err) {
      setPassword("");
      const f = failure(err);
      setFields(f.message ? { email: f.message } : f.fields);
    }
  }

  return (
    <form className="pe-card pe-stack" onSubmit={submit} noValidate aria-labelledby="change-email-heading">
      <h2 className="pe-card__title" id="change-email-heading">
        Change email address
      </h2>
      <p className="pe-note">Your email address also logs you in, so your current password is asked for.</p>
      <Field label="New email address" type="email" autoComplete="email" value={newEmail} onChange={setNewEmail} error={fields.email} />
      <PasswordField
        label="Current password"
        autoComplete="current-password"
        value={password}
        onChange={setPassword}
        error={fields.password}
        hint=" "
      />
      <div className="pe-actions">
        <button className="pe-btn pe-btn--secondary" type="button" onClick={() => onDone(false)}>
          Cancel
        </button>
        <button className="pe-btn pe-btn--primary" type="submit">
          Change email address
        </button>
      </div>
    </form>
  );
}

export default function ProfilePage() {
  const { address } = useSession();
  const profile = useOwnProfile();
  const [changing, setChanging] = useState(false);
  const [emailChanged, setEmailChanged] = useState(false);
  if (address?.kind !== "organization") return null;
  const p = profile.data;
  /* The names the visibility section offers are the saved ones, so it starts
     again when they change. */
  const savedNames = p ? [p.email, p.username, p.first_name, p.middle_name, p.last_name].join("|") : "";

  return (
    <SignedInShell>
      <div>
        <Link className="pe-link" to="/">
          Back to {address.organization.name}
        </Link>
      </div>
      <h1 className="pe-page-title">Your profile</h1>
      {emailChanged && <Notice>Your email address is changed. Use it to log in from now on.</Notice>}
      {p ? (
        <>
          <ProfileSection
            profile={p}
            onChange={() => {
              setChanging(true);
              setEmailChanged(false);
            }}
          />
          <VisibilitySection key={savedNames} profile={p} orgName={address.organization.name} />
          {changing && (
            <ChangeEmailSection
              onDone={(changed) => {
                setChanging(false);
                setEmailChanged(changed);
              }}
            />
          )}
        </>
      ) : profile.isError ? (
        <Alert>Your profile did not load. Reload the page to try again.</Alert>
      ) : null}
    </SignedInShell>
  );
}
