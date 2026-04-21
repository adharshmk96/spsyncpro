"use client"

import * as React from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { Building2Icon, LandmarkIcon } from "lucide-react"

import { TeamSwitcher } from "@/components/team-switcher"
import { NavUser } from "@/components/nav-user"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

const links = [
  {
    section: "Dashboard",
    links: [{ title: "Overview", to: "/dashboard/overview" }],
  },
  {
    section: "Jobs",
    links: [
      { title: "Backups", to: "/dashboard/backup-job/list" },
      { title: "Backup Schedules", to: "/dashboard/backup-job/schedules" },
      { title: "Restore", to: "/dashboard/restore-job/list" },
    ],
  },
  {
    section: "Settings",
    links: [{ title: "Organization", to: "/dashboard/organization/settings" }],
  },
] as const

const organizations = [
  {
    name: "SPSyncPro",
    logo: <Building2Icon />,
    plan: "v0.3.0",
  },
] as const

type SidebarUser = {
  name: string
  email: string
  avatar: string
}

export function AppSidebar({
  user,
  ...props
}: React.ComponentProps<typeof Sidebar> & { user: SidebarUser }) {
  const pathname = usePathname()

  return (
    <Sidebar {...props}>
      <SidebarHeader>
        <TeamSwitcher teams={[...organizations]} />
      </SidebarHeader>
      <SidebarContent>
        {links.map((section) => (
          <SidebarGroup key={section.section}>
            <SidebarGroupLabel>{section.section}</SidebarGroupLabel>
            <SidebarMenu>
              {section.links.map((link) => (
                <SidebarMenuItem key={link.to}>
                  <SidebarMenuButton asChild isActive={pathname === link.to}>
                    <Link href={link.to}>{link.title}</Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
    </Sidebar>
  )
}
