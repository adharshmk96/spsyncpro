import { LogoutButton } from "@/components/logout-button";
import { getCurrentMember } from "@/lib/api/session";

export default async function Page() {
  const member = await getCurrentMember();
  const displayName = member?.email ?? "there";

  return (
    <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
      <div className="rounded-xl border bg-card p-6">
        <div className="flex items-center justify-between gap-4">
          <p className="text-lg font-medium">Hello, {displayName}.</p>
          <LogoutButton />
        </div>
      </div>
    </div>
  );
}
