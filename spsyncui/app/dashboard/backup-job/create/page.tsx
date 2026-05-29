import { BackupJobForm } from "@/components/backup-job-form";

export default function DashboardBackupJobCreatePage() {
  return (
    <main className="p-6">
      <BackupJobForm mode="create" />
    </main>
  );
}
