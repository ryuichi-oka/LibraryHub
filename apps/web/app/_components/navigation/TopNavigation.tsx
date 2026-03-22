"use client";

import Link from "next/link";

import { USER_DEFAULT_PATH } from "../../_lib/authorization";
import styles from "./AppShell.module.css";

export type NavigationItem = {
  label: string;
  href: string;
};

type TopNavigationProps = {
  items: NavigationItem[];
  pathname: string;
  currentHash: string;
  onHashChange: (hash: string) => void;
};

function getHashFromTargetPath(targetPath: string): string {
  const [, targetHash = ""] = targetPath.split("#");
  return targetHash.length > 0 ? `#${targetHash}` : "";
}

function isActivePath(currentPath: string, currentHash: string, targetPath: string): boolean {
  const [targetBasePath, targetHash = ""] = targetPath.split("#");
  if (targetHash.length > 0) {
    return currentPath === targetBasePath && currentHash === `#${targetHash}`;
  }

  if (targetBasePath === USER_DEFAULT_PATH) {
    return currentPath === USER_DEFAULT_PATH && currentHash.length === 0;
  }

  return currentPath === targetBasePath || currentPath.startsWith(`${targetBasePath}/`);
}

export default function TopNavigation({ items, pathname, currentHash, onHashChange }: TopNavigationProps) {
  return (
    <nav className={styles.topNav} aria-label="主要メニュー">
      <ul className="m-0 flex list-none justify-center gap-2 px-5 pt-2 pb-3 sm:px-8">
        {items.map((item) => {
          const active = isActivePath(pathname, currentHash, item.href);
          return (
            <li key={item.href}>
              <Link
                className={`${styles.topNavLink} ${active ? styles.topNavLinkActive : ""}`}
                href={item.href}
                onClick={() => onHashChange(getHashFromTargetPath(item.href))}
              >
                {item.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
