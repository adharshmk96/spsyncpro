import Link from "next/link";

import { OrganizationForm } from "@/components/organization-form";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { serverApiFetch } from "@/lib/api/server";
import type { OrganizationResponse } from "@/lib/api/types";

export default async function DashboardOrganizationEditPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let organization: OrganizationResponse["organization"] | null = null;
  try {
    organization = (await serverApiFetch<OrganizationResponse>(`/organizations/${id}`)).organization;
  } catch (error) {
    console.error("Failed to load organization.", error);
  }

  if (!organization) {
    return (
      <main className="p-6">
        <Card className="mx-auto max-w-3xl p-6">
          <p className="text-sm text-destructive">Organization not found.</p>
          <Button asChild variant="outline" className="mt-4">
            <Link href="/dashboard/organization/list">Back to list</Link>
          </Button>
        </Card>
      </main>
    );
  }

  return (
    <main className="p-6">
      <OrganizationForm mode="edit" organization={organization} />
    </main>
  );
}
