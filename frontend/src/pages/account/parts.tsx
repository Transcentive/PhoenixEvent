import { useId, useState, type ReactNode } from "react";

/* The pieces the account screens share, drawn as in wireframes A1 to A7. */

export function Shell({ children, right }: { children: ReactNode; right?: ReactNode }) {
  return (
    <div className="pe-app">
      <header className="pe-topbar">
        <span className="pe-topbar__name">PhoenixEvent</span>
        {right}
      </header>
      <main className="pe-center">{children}</main>
    </div>
  );
}

export function Heading({ title, org }: { title: string; org?: ReactNode }) {
  return (
    <div className="pe-login__head">
      <h1 className="pe-login__title">{title}</h1>
      {org && <p className="pe-login__org">{org}</p>}
    </div>
  );
}

const WARNING = (
  <svg className="pe-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M12 4l9 16H3z" />
    <path d="M12 10v4M12 17v.5" />
  </svg>
);

const CLOCK = (
  <svg className="pe-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <circle cx="12" cy="12" r="9" />
    <path d="M12 7v5l3 2" />
  </svg>
);

export function Alert({ children }: { children: ReactNode }) {
  return (
    <div className="pe-alert" role="alert">
      {WARNING}
      <span>{children}</span>
    </div>
  );
}

export function Notice({ children }: { children: ReactNode }) {
  return (
    <div className="pe-notice" role="status">
      <span>{children}</span>
    </div>
  );
}

export function Badge({ role, children }: { role: "caution" | "critical"; children: ReactNode }) {
  return (
    <div>
      <span className={`pe-badge pe-badge--${role}`}>
        {role === "caution" ? CLOCK : WARNING}
        {children}
      </span>
    </div>
  );
}

interface FieldProps {
  label: string;
  hint?: ReactNode;
  error?: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
  autoComplete?: string;
  name?: string;
}

export function Field({ label, hint, error, value, onChange, type = "text", autoComplete, name }: FieldProps) {
  const id = useId();
  return (
    <div className={error ? "pe-field pe-field--error" : "pe-field"}>
      <label className="pe-field__label" htmlFor={id}>
        {label}
      </label>
      <input
        className="pe-input"
        id={id}
        name={name}
        type={type}
        autoComplete={autoComplete}
        value={value}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${id}-error` : undefined}
        onChange={(e) => onChange(e.target.value)}
      />
      {error && (
        <span className="pe-field__error" id={`${id}-error`}>
          {WARNING}
          {error}
        </span>
      )}
      {hint && <span className="pe-field__hint">{hint}</span>}
    </div>
  );
}

export function PasswordField({
  label,
  hint,
  error,
  value,
  onChange,
  autoComplete,
}: Omit<FieldProps, "type" | "name"> & { autoComplete: string }) {
  const id = useId();
  const [shown, setShown] = useState(false);
  return (
    <div className={error ? "pe-field pe-field--error" : "pe-field"}>
      <label className="pe-field__label" htmlFor={id}>
        {label}
      </label>
      <div className="pe-password">
        <input
          className="pe-input"
          id={id}
          type={shown ? "text" : "password"}
          autoComplete={autoComplete}
          value={value}
          aria-invalid={error ? true : undefined}
          onChange={(e) => onChange(e.target.value)}
        />
        <button className="pe-btn pe-btn--secondary" type="button" aria-controls={id} aria-pressed={shown} onClick={() => setShown(!shown)}>
          {shown ? "Hide" : "Show"}
        </button>
      </div>
      {error && (
        <span className="pe-field__error">
          {WARNING}
          {error}
        </span>
      )}
      <span className="pe-field__hint" role="status">
        {hint ?? (shown ? "Password is visible." : "Password is hidden.")}
      </span>
    </div>
  );
}

export const PASSWORD_HINT = "12 to 64 characters. Spaces are allowed. A common password is refused.";

export function Note({ children }: { children: ReactNode }) {
  return <p className="pe-note">{children}</p>;
}

export function Text({ children }: { children: ReactNode }) {
  return <p className="pe-text">{children}</p>;
}
