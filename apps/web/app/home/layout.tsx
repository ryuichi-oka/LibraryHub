import { ReactNode } from "react";

import AppShell from "../_components/navigation/AppShell";

type HomeLayoutProps = {
  children: ReactNode;
};

export default function HomeLayout({ children }: HomeLayoutProps) {
  return <AppShell>{children}</AppShell>;
}
