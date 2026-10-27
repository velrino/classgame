"use client";

import type { User } from "firebase/auth";
import { ChevronLeft, ChevronRight, LogOut } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { useEffect, useMemo, useState, useSyncExternalStore } from "react";

import { SkipToMain } from "@/components/skip-to-main";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
import { getCookie } from "@/lib/cookies";
import { cn } from "@/lib/utils";
import type { TeacherClassDto } from "@/modules/classrooms";

import { rememberSelectedClass } from "@/modules/classrooms";
import {
  dashboardSidebarData,
  type DashboardNavItem,
} from "@/modules/dashboard/adapters/types/navigation";

type DashboardShellProps = {
  readonly children: ReactNode;
  readonly teacherClasses?: TeacherClassDto[];
  readonly user: User;
  readonly onSignOut: () => void;
};

export function DashboardShell({
  children,
  onSignOut,
  teacherClasses = [],
  user,
}: DashboardShellProps) {
  const [activeHash, setActiveHash] = useState("#overview");
  const persistedSidebarOpen = useSyncExternalStore(
    subscribeToSidebarPreference,
    readSidebarDefaultOpen,
    readServerSidebarDefaultOpen,
  );
  const [sidebarOpenOverride, setSidebarOpenOverride] = useState<
    boolean | null
  >(null);
  const sidebarOpen = sidebarOpenOverride ?? persistedSidebarOpen;
  const pathname = usePathname();

  useEffect(() => {
    function syncActiveHash() {
      setActiveHash(window.location.hash || "#overview");
    }

    syncActiveHash();
    window.addEventListener("hashchange", syncActiveHash);

    return () => {
      window.removeEventListener("hashchange", syncActiveHash);
    };
  }, []);

  return (
    <main className="dashboard-theme min-h-svh bg-sidebar text-foreground">
      <SidebarProvider open={sidebarOpen} onOpenChange={setSidebarOpenOverride}>
        <SkipToMain />
        <DashboardAppSidebar
          activeHash={activeHash}
          pathname={pathname}
          teacherClasses={teacherClasses}
          user={user}
          onSignOut={onSignOut}
        />
        <SidebarInset className="@container/content md:peer-data-[variant=inset]:ml-2">
          <DashboardHeader user={user} />
          {children}
        </SidebarInset>
      </SidebarProvider>
    </main>
  );
}

function DashboardAppSidebar({
  activeHash,
  onSignOut,
  pathname,
  teacherClasses,
  user,
}: {
  readonly activeHash: string;
  readonly onSignOut: () => void;
  readonly pathname: string;
  readonly teacherClasses: TeacherClassDto[];
  readonly user: User;
}) {
  const navGroups = useMemo(
    () =>
      dashboardSidebarData.navGroups.map((group) => ({
        ...group,
        items: group.items.map((item) => {
          const scopedItem = {
            ...item,
            url: resolveScopedDashboardURL(item.url, pathname),
          };

          return item.title === "Classes"
            ? {
                ...scopedItem,
                items: teacherClasses.map((teacherClass) => ({
                  classId: teacherClass.class_id,
                  title:
                    teacherClass.class_name.trim() || teacherClass.class_code,
                  url: `/dashboard/class/${encodeURIComponent(teacherClass.class_id)}`,
                })),
              }
            : scopedItem;
        }),
      })),
    [pathname, teacherClasses],
  );

  return (
    <Sidebar collapsible="icon" variant="inset">
      <SidebarHeader>
        <Link
          href="/dashboard"
          aria-label={dashboardSidebarData.team.name}
          className="flex h-10 w-full items-center justify-center rounded-lg bg-white px-3 font-display text-sm font-bold text-primary transition-[width,height,padding] group-data-[collapsible=icon]:mx-auto group-data-[collapsible=icon]:size-8 group-data-[collapsible=icon]:bg-primary group-data-[collapsible=icon]:px-0 group-data-[collapsible=icon]:text-primary-foreground"
        >
          <span className="truncate text-xl text-primary group-data-[collapsible=icon]:hidden">
            {dashboardSidebarData.team.name}
          </span>
          <span className="hidden group-data-[collapsible=icon]:block">
            {dashboardSidebarData.team.name.charAt(0)}
          </span>
        </Link>
      </SidebarHeader>

      <SidebarContent>
        {navGroups.map((group) => (
          <SidebarGroup key={group.title}>
            <SidebarGroupLabel>{titleCase(group.title)}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => (
                  <DashboardSidebarItem
                    key={`${group.title}-${item.title}`}
                    activeHash={activeHash}
                    item={item}
                    pathname={pathname}
                  />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>

      <SidebarFooter>
        <SidebarUser user={user} onSignOut={onSignOut} />
      </SidebarFooter>
      <DashboardSidebarTrigger />
      <SidebarRail />
    </Sidebar>
  );
}

function DashboardSidebarTrigger() {
  const { state, toggleSidebar } = useSidebar();
  const isExpanded = state === "expanded";

  return (
    <Button
      type="button"
      size="icon"
      aria-label={isExpanded ? "Collapse sidebar" : "Expand sidebar"}
      title={isExpanded ? "Collapse sidebar" : "Expand sidebar"}
      className="absolute right-2 top-1/2 z-30 hidden size-6 -translate-y-1/2 translate-x-1/2 rounded-full border-2 border-sidebar bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 md:inline-flex"
      onClick={toggleSidebar}
    >
      {isExpanded ? (
        <ChevronLeft className="size-3.5" aria-hidden />
      ) : (
        <ChevronRight className="size-3.5" aria-hidden />
      )}
    </Button>
  );
}

function DashboardSidebarItem({
  activeHash,
  item,
  pathname,
}: {
  readonly activeHash: string;
  readonly item: DashboardNavItem;
  readonly pathname: string;
}) {
  const { setOpenMobile } = useSidebar();
  const Icon = item.icon;
  const hasActiveChild = item.items?.some((subItem) =>
    isActiveDashboardRoute(subItem.url, pathname, activeHash),
  );

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        asChild
        aria-disabled={item.disabled}
        isActive={
          isActiveDashboardRoute(item.url, pathname, activeHash) ||
          hasActiveChild
        }
        tooltip={item.title}
      >
        <Link
          href={item.disabled ? "#" : item.url}
          onClick={(event) => {
            if (item.disabled) {
              event.preventDefault();
              return;
            }

            setOpenMobile(false);
          }}
        >
          <Icon aria-hidden />
          <span>{item.title}</span>
          {item.badge ? (
            <SidebarMenuBadge>{item.badge}</SidebarMenuBadge>
          ) : null}
        </Link>
      </SidebarMenuButton>
      {item.items?.length ? (
        <ul className="mx-3 flex min-w-0 flex-col gap-1 border-l border-sidebar-border px-2 py-1 group-data-[collapsible=icon]:hidden">
          {item.items.map((subItem) => (
            <li key={subItem.classId} className="min-w-0">
              <Link
                href={subItem.url}
                className={cn(
                  "block truncate rounded-md px-2 py-1.5 text-xs text-sidebar-foreground/75 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                  isActiveDashboardRoute(subItem.url, pathname, activeHash) &&
                    "bg-sidebar-accent font-medium text-sidebar-accent-foreground",
                )}
                title={subItem.title}
                onClick={() => {
                  rememberSelectedClass(subItem.classId);
                  setOpenMobile(false);
                }}
              >
                {subItem.title}
              </Link>
            </li>
          ))}
        </ul>
      ) : null}
    </SidebarMenuItem>
  );
}

function DashboardHeader({ user }: { readonly user: User }) {
  return (
    <header className="z-50 h-16">
      <div className="relative flex h-full items-center gap-3 p-4 sm:gap-4">
        <SidebarTrigger className="max-md:scale-125 md:hidden" />
        <Separator orientation="vertical" className="h-6 md:hidden" />

        <div className="ml-auto flex min-w-0 items-center gap-2">
          <TeacherAvatar user={user} />
        </div>
      </div>
    </header>
  );
}

function SidebarUser({
  onSignOut,
  user,
}: {
  readonly onSignOut: () => void;
  readonly user: User;
}) {
  const displayName = resolveTeacherDisplayName(user);
  const email = user.email ?? "Teacher account";

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton size="lg" tooltip={displayName} onClick={onSignOut}>
          <UserAvatar user={user} className="rounded-lg" />
          <div className="grid flex-1 text-start text-sm leading-tight">
            <span className="truncate font-semibold">{displayName}</span>
            <span className="truncate text-xs">{email}</span>
          </div>
          <LogOut className="ml-auto size-4" aria-hidden />
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

function TeacherAvatar({ user }: { readonly user: User }) {
  return (
    <Button
      type="button"
      variant="ghost"
      className="relative h-8 w-8 rounded-full p-0"
    >
      <UserAvatar user={user} />
      <span className="sr-only">Teacher account</span>
    </Button>
  );
}

function UserAvatar({
  className,
  user,
}: {
  readonly className?: string;
  readonly user: User;
}) {
  const displayName = resolveTeacherDisplayName(user);

  if (user.photoURL) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={user.photoURL}
        alt=""
        className={cn("size-8 rounded-full object-cover", className)}
      />
    );
  }

  return (
    <span
      className={cn(
        "grid size-8 place-items-center rounded-full bg-muted text-sm font-semibold",
        className,
      )}
    >
      {displayName.charAt(0).toUpperCase()}
    </span>
  );
}

function readSidebarDefaultOpen() {
  return getCookie("sidebar_state") !== "false";
}

function readServerSidebarDefaultOpen() {
  return true;
}

function subscribeToSidebarPreference() {
  return () => undefined;
}

function resolveTeacherDisplayName(user: User) {
  return user.displayName ?? user.email ?? "Teacher";
}

function titleCase(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1).toLowerCase();
}

function isActiveDashboardRoute(
  target: string,
  pathname: string,
  activeHash: string,
) {
  const route = parseDashboardRoute(target);
  if (route.pathname !== pathname) {
    return false;
  }

  if (!route.hash) {
    return activeHash === "" || activeHash === "#overview";
  }

  return route.hash === activeHash;
}

function parseDashboardRoute(target: string) {
  if (target.startsWith("#")) {
    return {
      pathname: "/dashboard",
      hash: target,
    };
  }

  const [pathname, hash] = target.split("#");

  return {
    pathname: pathname || "/dashboard",
    hash: hash ? `#${hash}` : "",
  };
}

function resolveScopedDashboardURL(target: string, pathname: string) {
  if (!pathname.startsWith("/dashboard/class/")) {
    return target;
  }

  const route = parseDashboardRoute(target);
  if (route.pathname !== "/dashboard") {
    return target;
  }

  return `${pathname}${route.hash || "#overview"}`;
}
