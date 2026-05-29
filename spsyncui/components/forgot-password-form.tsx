"use client";

import Link from "next/link";
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

export function ForgotPasswordForm({ className, ...props }: React.ComponentProps<"div">) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitted, setSubmitted] = useState(false);

  const handleSubmit = async (formData: FormData) => {
    const email = formData.get("email");
    if (typeof email !== "string" || email.length === 0) {
      setError("Email is required.");
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      await clientApiFetch("/forgot-password", {
        method: "POST",
        body: JSON.stringify({ email }),
      });
      setSubmitted(true);
    } catch (submitError) {
      console.warn("Forgot-password request failed:", submitError);
      setError(toErrorMessage(submitError, "Unable to send reset link."));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className={cn("flex flex-col gap-6", className)} {...props}>
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-xl">Forgot your password?</CardTitle>
          <CardDescription>
            Enter your email and we&apos;ll send you a reset link.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {submitted ? (
            <p className="text-sm text-muted-foreground">
              If an account exists for that email, a password reset link has been sent.
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
                    placeholder="m@example.com"
                    required
                  />
                  {error ? (
                    <FieldDescription className="text-destructive">{error}</FieldDescription>
                  ) : null}
                </Field>
                <Field>
                  <Button type="submit" disabled={isLoading}>
                    {isLoading ? "Sending..." : "Send reset link"}
                  </Button>
                  <FieldDescription className="text-center">
                    Remembered it? <Link href="/login">Back to login</Link>
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
