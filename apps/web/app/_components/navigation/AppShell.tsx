"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { ReactNode, useEffect, useMemo, useState } from "react";

import { AuthSession, clearAuthSession, isSessionExpired, loadAuthSession } from "../../_lib/authSession";
import { LOGIN_PATH, USER_DEFAULT_PATH, canAccessPath, isAdminRole } from "../../_lib/authorization";
import SearchPanel from "./SearchPanel";
import TopNavigation, { NavigationItem } from "./TopNavigation";
import styles from "./AppShell.module.css";

const PRIMARY_NAV_ITEMS: NavigationItem[] = [
  { label: "新着書籍一覧", href: USER_DEFAULT_PATH },
  { label: "おすすめ", href: "/home#recommendations" },
];

function getHamburgerItems(role: string): NavigationItem[] {
  if (isAdminRole(role)) {
    return [{ label: "管理者画面（利用者管理）", href: "/admin/users/status" }];
  }

  return [];
}

type AppShellProps = {
  children: ReactNode;
};

export default function AppShell({ children }: AppShellProps) {
  const router = useRouter();
  const pathname = usePathname();
  const [session, setSession] = useState<AuthSession | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [currentHash, setCurrentHash] = useState("");

  useEffect(() => {
    setIsReady(false);

    const storedSession = loadAuthSession();
    if (!storedSession) {
      router.replace(LOGIN_PATH);
      return;
    }

    if (isSessionExpired(storedSession.expiresAt)) {
      clearAuthSession();
      router.replace(LOGIN_PATH);
      return;
    }

    const currentPath = pathname || USER_DEFAULT_PATH;
    if (!canAccessPath(storedSession.role, currentPath)) {
      router.replace(USER_DEFAULT_PATH);
      return;
    }

    setSession(storedSession);
    setIsReady(true);
  }, [pathname, router]);

  useEffect(() => {
    setIsMenuOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const syncHash = () => setCurrentHash(window.location.hash);
    syncHash();
    window.addEventListener("hashchange", syncHash);
    return () => window.removeEventListener("hashchange", syncHash);
  }, [pathname]);

  const navItems = useMemo(() => {
    return PRIMARY_NAV_ITEMS;
  }, []);
  const hamburgerItems = useMemo(() => {
    return getHamburgerItems(session?.role ?? "");
  }, [session?.role]);

  function handleLogout() {
    clearAuthSession();
    router.replace(LOGIN_PATH);
  }

  if (!isReady || !session) {
    return null;
  }
  const isAdmin = isAdminRole(session.role);
  const currentPath = pathname || USER_DEFAULT_PATH;
  const isHomePath = currentPath === USER_DEFAULT_PATH;

  return (
    <div className={`${styles.shell} bg-slate-50`}>
      <header className={styles.header}>
        <div className="relative flex w-full items-center justify-between gap-3 px-5 py-2 sm:px-8">
          <Link href={USER_DEFAULT_PATH} className={styles.brandLink} aria-label="トップ画面へ移動">
            <p className={styles.brand}>LIBRARYHUB</p>
            <p className={styles.tagline}>図書管理システム</p>
          </Link>

          <div className="flex items-center gap-2">
            {isAdmin && <span className={styles.roleBadge}>管理者</span>}
            <button type="button" className={styles.logoutButton} onClick={handleLogout} aria-label="ログアウト">
              <span className={styles.logoutIcon} aria-hidden="true">
                ↪
              </span>
            </button>
            <button
              type="button"
              className={styles.menuButton}
              aria-label="メニュー"
              aria-expanded={isMenuOpen}
              aria-controls="header-menu"
              onClick={() => setIsMenuOpen((prev) => !prev)}
            >
              <span className={styles.menuIcon} aria-hidden="true">
                ≡
              </span>
            </button>
          </div>

          {isMenuOpen && (
            <div id="header-menu" className={styles.menuPanel}>
              <ul className="m-0 list-none p-0">
                {hamburgerItems.map((item) => (
                  <li key={item.href}>
                    <Link className={styles.menuLink} href={item.href} onClick={() => setIsMenuOpen(false)}>
                      {item.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>

        {isHomePath ? (
          <TopNavigation
            items={navItems}
            pathname={pathname || ""}
            currentHash={currentHash}
            onHashChange={setCurrentHash}
          />
        ) : null}
      </header>

      <div className="flex w-full flex-col gap-4 px-5 py-5 sm:px-8 md:flex-row md:items-start">
        {isHomePath ? <SearchPanel /> : null}

        <main className="min-w-0 flex-1">{children}</main>
      </div>
    </div>
  );
}
