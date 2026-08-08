import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, UserCircle } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { listUsers, setUserDisabled } from "../api/users";
import { ApiError, type User } from "../api/types";

export function UsersListPage() {
  const queryClient = useQueryClient();
  const [toggleError, setToggleError] = useState<string | null>(null);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["users"],
    queryFn: () => listUsers(),
  });

  const toggleMutation = useMutation({
    mutationFn: (vars: { userId: number; disabled: boolean }) => setUserDisabled(vars.userId, vars.disabled),
    onSuccess: () => {
      setToggleError(null);
      queryClient.invalidateQueries({ queryKey: ["users"] });
    },
    onError: (err) => {
      setToggleError(err instanceof ApiError ? err.message : "Couldn't update this user. Try again.");
    },
  });

  if (isError) {
    return <ErrorState onRetry={() => refetch()} />;
  }

  const columns: Column<User>[] = [
    {
      header: "Email",
      render: (u) => (
        <div className="flex items-center gap-3">
          <UserCircle size={16} className="text-on-surface-variant" />
          <span className="text-on-surface">{u.email}</span>
        </div>
      ),
    },
    {
      header: "Role",
      render: (u) => (
        <span className="capitalize">{u.role}{u.role === "restricted" && u.site_id ? ` · ${u.site_id}` : ""}</span>
      ),
    },
    { header: "Created", isMono: true, render: (u) => new Date(u.created_at).toLocaleDateString() },
    {
      header: "Status",
      render: (u) => (u.disabled_at ? <span className="text-error">Disabled</span> : <span className="text-success">Active</span>),
    },
    {
      header: "",
      align: "right",
      render: (u) => (
        <button
          type="button"
          disabled={toggleMutation.isPending}
          onClick={(e) => {
            e.stopPropagation();
            const nextDisabled = !u.disabled_at;
            if (window.confirm(`${nextDisabled ? "Disable" : "Re-enable"} ${u.email}?`)) {
              toggleMutation.mutate({ userId: u.id, disabled: nextDisabled });
            }
          }}
          className="text-[12px] text-on-surface-variant hover:text-primary transition-colors disabled:opacity-60"
        >
          {u.disabled_at ? "Enable" : "Disable"}
        </button>
      ),
    },
  ];

  return (
    <>
      <TopNav title="Users & Roles" />
      <div className="flex-1 p-grid-margin space-y-6">
        <div className="flex justify-end">
          <Link
            to="/app/users/invite"
            className="bg-primary hover:opacity-90 text-on-primary font-semibold py-2.5 px-5 rounded-full flex items-center gap-2 transition-all shadow-soft"
          >
            <Plus size={18} />
            <span>Invite User</span>
          </Link>
        </div>

        {toggleError && <p className="font-label-caps text-label-caps text-error">{toggleError}</p>}

        {isLoading || !data ? (
          <div className="h-64 glass-card rounded-xl animate-pulse" />
        ) : data.items.length === 0 ? (
          <EmptyState title="No users yet" body="Invite your first user to give them dashboard access." />
        ) : (
          <DataTable columns={columns} rows={data.items} rowKey={(u) => String(u.id)} />
        )}
      </div>
    </>
  );
}
