export const ADMIN_ONLY_PATH_PREFIX = "/admin";
export const ADMIN_DEFAULT_PATH = "/admin/users/status";
export const USER_DEFAULT_PATH = "/home";
export const LOGIN_PATH = "/";

export function isAdminRole(role: string): boolean {
  return role === "ADMIN";
}

// ロールと遷移先URLから、遷移可否を判定する。
export function canAccessPath(role: string, pathname: string): boolean {
  if (pathname.startsWith(ADMIN_ONLY_PATH_PREFIX)) {
    return isAdminRole(role);
  }
  return true;
}

// ログイン直後に遷移すべき既定画面を返す。
export function getDefaultPathByRole(role: string): string {
  // ログイン直後はロールに関係なくトップ画面へ遷移する。
  // 管理者向け画面へのアクセス可否は canAccessPath で別途制御する。
  return USER_DEFAULT_PATH;
}
