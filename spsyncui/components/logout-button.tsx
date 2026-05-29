"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { logout } from "@/lib/api/auth";

export function LogoutButton() {
  const router = useRouter();
  const [isLoading, setIsLoading] = useState(false);

  const handleLogout = async () => {
    setIsLoading(true);

    try {
      await logout();
      router.push("/login");
      router.refresh();
    } catch (logoutError) {
      console.error("Unexpected sign-out error:", logoutError);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Button onClick={handleLogout} disabled={isLoading}>
      {isLoading ? "Logging out..." : "Logout"}
    </Button>
  );
}
