import { redirect } from "next/navigation";

import { AppSidebar } from "@/components/app-sidebar";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ModeToggle } from "@/components/mode-toggle";
import { getCurrentMember } from "@/lib/api/session";

export default async function DashboardLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const member = await getCurrentMember();

  if (!member) {
    redirect("/login");
  }

  const displayEmail = member.email;
  const displayName = displayEmail.split("@")[0] ?? "User";

  return (
    <TooltipProvider>
      <SidebarProvider>
        <AppSidebar
          user={{
            name: displayName,
            email: displayEmail,
            avatar: "",
          }}
        />
        <SidebarInset>
          <header className="flex h-16 w-full shrink-0 items-center justify-between gap-4 px-4">
            <div className="flex min-w-0 flex-1 items-center gap-2">
              <SidebarTrigger className="md:hidden" />
              <Breadcrumb>
                <BreadcrumbList>
                  <BreadcrumbItem className="hidden md:block">
                    <BreadcrumbLink href="/">SPSyncPro</BreadcrumbLink>
                  </BreadcrumbItem>
                  <BreadcrumbSeparator className="hidden md:block" />
                  <BreadcrumbItem>
                    <BreadcrumbPage>Dashboard</BreadcrumbPage>
                  </BreadcrumbItem>
                </BreadcrumbList>
              </Breadcrumb>
            </div>
            <ModeToggle />
          </header>
          {children}
        </SidebarInset>
      </SidebarProvider>
    </TooltipProvider>
  );
}
