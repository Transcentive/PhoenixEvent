import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, getMyRoles, getPersonRoles, setPersonRoles, type OrganizationRole } from "@/api/client";
import { useSession } from "@/session";
import { Alert } from "@/pages/account/parts";

/* The organization roles the signed-in account holds. The page shows each
   action only to the roles 7B.6 names; the server checks again. */
export function useMyRoles() {
  const { session } = useSession();
  return useQuery({
    queryKey: ["myRoles", session?.account.id],
    queryFn: () => getMyRoles(session!.access_token),
    enabled: !!session,
  });
}

export function useIsAdministrator() {
  return useMyRoles().data?.includes("organization_administrator") ?? false;
}

const CHOICES: [OrganizationRole, string][] = [
  ["organization_administrator", "Organization Administrator"],
  ["organization_leadership", "Organization Leadership"],
];

/* D5, Change roles (7B.4, 7B.6). Member is every person, so it is always
   ticked and cannot be cleared. */
export function ChangeRoles({ personId, onClose }: { personId: string; onClose: () => void }) {
  const { address, session } = useSession();
  const queryClient = useQueryClient();
  const current = useQuery({
    queryKey: ["personRoles", personId],
    queryFn: () => getPersonRoles(session!.access_token, personId),
    enabled: !!session,
    retry: false,
  });
  const [chosen, setChosen] = useState<OrganizationRole[] | null>(null);
  const roles = chosen ?? current.data?.roles ?? [];
  const save = useMutation({
    mutationFn: () => setPersonRoles(session!.access_token, personId, roles),
    onSuccess: async () => {
      await queryClient.invalidateQueries();
      onClose();
    },
  });
  const first = useRef<HTMLInputElement>(null);
  useEffect(() => {
    first.current?.focus();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose, current.data]);
  if (address?.kind !== "organization") return null;
  const orgName = address.organization.name;
  const error = save.error instanceof ApiError ? (save.error.fields[0]?.message ?? save.error.message) : null;

  function toggle(role: OrganizationRole, on: boolean) {
    setChosen(on ? [...roles, role] : roles.filter((r) => r !== role));
  }

  return (
    <div className="pe-scrim pe-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="pe-dialog" role="dialog" aria-modal="true" aria-labelledby="change-roles-title">
        <h1 className="pe-dialog__title" id="change-roles-title">
          Change roles for {current.data?.name ?? "…"}
        </h1>
        {current.isError ? (
          <Alert>These roles did not load. Close this and try again.</Alert>
        ) : (
          <>
            <p className="pe-dialog__body">Roles here apply inside {orgName} only.</p>
            {error && <Alert>{error}</Alert>}
            <fieldset className="pe-choices">
              <legend className="pe-field__label">Organization roles</legend>
              <label className="pe-choice">
                <input type="checkbox" checked disabled ref={first} />
                Member
              </label>
              {CHOICES.map(([role, label]) => (
                <label key={role} className="pe-choice">
                  <input
                    type="checkbox"
                    checked={roles.includes(role)}
                    disabled={!current.data}
                    onChange={(e) => toggle(role, e.target.checked)}
                  />
                  {label}
                </label>
              ))}
            </fieldset>
            <p className="pe-dialog__body pe-dialog__note">
              Event and station roles, such as Event Chair and Station Chair, are set on the event board and station
              roster screens. Payments are handled on the Organization payment screen.
            </p>
            <p className="pe-dialog__body pe-dialog__note">
              The organization has three roles: Member, Organization Leadership and Organization Administrator. Every
              person in the organization is a Member. What each role may do is set in section 7B.6 of the
              specification.
            </p>
          </>
        )}
        <div className="pe-dialog__actions">
          <button className="pe-btn pe-btn--secondary" type="button" onClick={onClose}>
            Cancel
          </button>
          <button
            className="pe-btn pe-btn--primary"
            type="button"
            disabled={!current.data || save.isPending}
            onClick={() => save.mutate()}
          >
            Save roles
          </button>
        </div>
      </div>
    </div>
  );
}
