import type { LucideIcon } from "lucide-react";
import {
  BookOpenCheck,
  Gamepad2,
  LayoutDashboard,
  Network,
  Settings,
} from "lucide-react";

export type DashboardNavItem = {
  title: string;
  url: string;
  icon: LucideIcon;
  badge?: string;
  disabled?: boolean;
  items?: DashboardNavSubItem[];
};

export type DashboardNavSubItem = {
  title: string;
  url: string;
  classId: string;
};

export type DashboardNavGroup = {
  title: string;
  items: DashboardNavItem[];
};

export type DashboardSidebarData = {
  team: {
    name: string;
    plan: string;
    logo: LucideIcon;
  };
  navGroups: DashboardNavGroup[];
};

export const dashboardSidebarData: DashboardSidebarData = {
  team: {
    name: "ClassGame",
    plan: "Teacher workspace",
    logo: Gamepad2,
  },
  navGroups: [
    {
      title: "General",
      items: [
        {
          title: "Overview",
          url: "/dashboard",
          icon: LayoutDashboard,
        },
        {
          title: "Architecture",
          url: "/dashboard/architecture",
          icon: Network,
        },
        {
          title: "Classes",
          url: "/dashboard/classes",
          icon: BookOpenCheck,
        },
      ],
    },
    {
      title: "System",
      items: [
        {
          title: "Settings",
          url: "/dashboard#settings",
          icon: Settings,
          disabled: true,
        },
      ],
    },
  ],
};

export const dashboardTopNav = [
  {
    title: "Overview",
    href: "/dashboard",
    isActive: true,
    disabled: false,
  },
  {
    title: "Classes",
    href: "/dashboard/classes",
    isActive: false,
    disabled: false,
  },
  {
    title: "Missions",
    href: "/dashboard#mission-builder",
    isActive: false,
    disabled: false,
  },
  {
    title: "Students",
    href: "/dashboard#students",
    isActive: false,
    disabled: false,
  },
  {
    title: "Teacher notes",
    href: "/dashboard#notebook",
    isActive: false,
    disabled: false,
  },
];
