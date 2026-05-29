"use client";

import { FormEvent, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";

export default function DashboardAccountPage() {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);

    if (newPassword !== confirmPassword) {
      setErrorMessage("New passwords do not match.");
      return;
    }
    if (newPassword.length < 8) {
      setErrorMessage("New password must be at least 8 characters long.");
      return;
    }

    setIsSaving(true);
    try {
      await clientApiFetch("/change-password", {
        method: "POST",
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });
      setSuccessMessage("Password changed successfully.");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (error) {
      console.error("Change password failed.", error);
      setErrorMessage(toErrorMessage(error, "Failed to change password."));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <main className="p-6">
      <Card className="mx-auto max-w-xl p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold">Account</h1>
          <p className="mt-1 text-sm text-muted-foreground">Change your account password.</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="currentPassword">Current password</Label>
            <Input
              id="currentPassword"
              type="password"
              value={currentPassword}
              onChange={(event) => setCurrentPassword(event.target.value)}
              required
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="newPassword">New password</Label>
            <Input
              id="newPassword"
              type="password"
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
              required
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="confirmPassword">Confirm new password</Label>
            <Input
              id="confirmPassword"
              type="password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              required
            />
          </div>

          <div className="flex items-center gap-3 pt-2">
            <Button type="submit" disabled={isSaving}>
              {isSaving ? "Saving..." : "Change password"}
            </Button>
          </div>

          {successMessage ? (
            <p className="text-sm text-emerald-600 dark:text-emerald-500">{successMessage}</p>
          ) : null}
          {errorMessage ? <p className="text-sm text-destructive">{errorMessage}</p> : null}
        </form>
      </Card>
    </main>
  );
}
