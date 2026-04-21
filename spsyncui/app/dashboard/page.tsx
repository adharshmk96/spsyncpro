import { headers } from "next/headers";

import { LogoutButton } from "@/components/logout-button";
import { auth } from "@/lib/auth";

export default async function Page() {
  const session = await auth.api.getSession({
    headers: await headers(),
  });
  const username = session?.user?.name ?? "User";

  return (
    <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
      <div className="rounded-xl border bg-card p-6">
        <div className="flex items-center justify-between gap-4">
          <p className="text-lg font-medium">Hello, {username}.</p>
          <LogoutButton />
        </div>
      </div>
    </div>
  );
}
