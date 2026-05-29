"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

import { cn } from "@/lib/utils";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export function ResetPasswordForm({ className, ...props }: React.ComponentProps<"div">) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const emailFromLink = searchParams.get("email") ?? "";
  const tokenFromLink = searchParams.get("token") ?? "";

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (formData: FormData) => {
    const email = (formData.get("email") as string | null) ?? "";
    const token = (formData.get("token") as string | null) ?? "";
    const password = (formData.get("password") as string | null) ?? "";
    const confirmPassword = (formData.get("confirmPassword") as string | null) ?? "";

    if (!email || !token) {
      setError("This reset link is missing required information.");
      return;
    }
    if (password !== confirmPassword) {
      setError("Passwords do not match.");
      return;
    }
    if (password.length < 8) {
      setError("Password must be at least 8 characters long.");
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      await clientApiFetch("/reset-password", {
        method: "POST",
        body: JSON.stringify({ email, token, password }),
      });
      setSuccess(true);
      setTimeout(() => router.push("/login"), 1500);
    } catch (submitError) {
      console.warn("Reset-password request failed:", submitError);
      setError(toErrorMessage(submitError, "Unable to reset password."));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className={cn("flex flex-col gap-6", className)} {...props}>
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-xl">Reset your password</CardTitle>
          <CardDescription>Choose a new password for your account.</CardDescription>
        </CardHeader>
        <CardContent>
          {success ? (
            <p className="text-sm text-emerald-600 dark:text-emerald-500">
              Password reset successfully. Redirecting to login...
            </p>
          ) : (
            <form action={handleSubmit}>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="email">Email</FieldLabel>
                  <Input
                    id="email"
                    name="email"
                    type="email"
                    defaultValue={emailFromLink}
                    required
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="token">Reset token</FieldLabel>
                  <Input id="token" name="token" type="text" defaultValue={tokenFromLink} required />
                </Field>
                <Field>
                  <FieldLabel htmlFor="password">New password</FieldLabel>
                  <Input id="password" name="password" type="password" required />
                </Field>
                <Field>
                  <FieldLabel htmlFor="confirmPassword">Confirm new password</FieldLabel>
                  <Input id="confirmPassword" name="confirmPassword" type="password" required />
                  <FieldDescription>Must be at least 8 characters long.</FieldDescription>
                  {error ? (
                    <FieldDescription className="text-destructive">{error}</FieldDescription>
                  ) : null}
                </Field>
                <Field>
                  <Button type="submit" disabled={isLoading}>
                    {isLoading ? "Resetting..." : "Reset password"}
                  </Button>
                  <FieldDescription className="text-center">
                    <Link href="/login">Back to login</Link>
                  </FieldDescription>
                </Field>
              </FieldGroup>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
