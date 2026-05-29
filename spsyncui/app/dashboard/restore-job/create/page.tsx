import { RestoreJobForm } from "@/components/restore-job-form";

export default function DashboardRestoreJobCreatePage() {
  return (
    <main className="p-6">
      <RestoreJobForm mode="create" />
    </main>
  );
}
