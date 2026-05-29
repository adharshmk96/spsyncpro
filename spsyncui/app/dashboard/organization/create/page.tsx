import { OrganizationForm } from "@/components/organization-form";

export default function DashboardOrganizationCreatePage() {
  return (
    <main className="p-6">
      <OrganizationForm mode="create" />
    </main>
  );
}
