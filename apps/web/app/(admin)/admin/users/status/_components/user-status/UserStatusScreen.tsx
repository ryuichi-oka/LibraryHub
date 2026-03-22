"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { clearAuthSession, isSessionExpired, loadAuthSession } from "../../../../../../_lib/authSession";
import styles from "./UserStatusScreen.module.css";
import UserStatusFeedback from "./UserStatusFeedback";
import UserStatusForm from "./UserStatusForm";
import { AdminUser, AdminUsersResponse, ErrorResponse, UpdateUserStatusResponse, UpdatedUser, UserStatus } from "./types";

function formatErrorMessage(status: number, fallback: string): string {
  if (status === 400) {
    return "利用状態の指定が正しくありません。もう一度選択してください。";
  }
  if (status === 401) {
    return "認証に失敗しました。再ログインしてください。";
  }
  if (status === 403) {
    return "管理者権限が必要です。";
  }
  if (status === 404) {
    return "指定したユーザーが見つかりません。";
  }
  if (status === 409) {
    return "自分自身の利用状態は変更できません。";
  }
  return fallback || "ステータス更新に失敗しました。時間をおいて再試行してください。";
}

function decodeBase64URL(value: string): string {
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
  return atob(padded);
}

function extractUserIDFromToken(token: string): string {
  try {
    const parts = token.split(".");
    if (parts.length < 2) {
      return "";
    }
    const payload = JSON.parse(decodeBase64URL(parts[1])) as { sub?: unknown };
    return typeof payload.sub === "string" ? payload.sub : "";
  } catch {
    return "";
  }
}

export default function UserStatusScreen() {
  const [adminToken, setAdminToken] = useState("");
  const [adminUserID, setAdminUserID] = useState("");
  const [isSessionReady, setIsSessionReady] = useState(false);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [selectedUserID, setSelectedUserID] = useState("");
  const [status, setStatus] = useState<UserStatus>("INACTIVE");
  const [isFetchingUsers, setIsFetchingUsers] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [updatedUser, setUpdatedUser] = useState<UpdatedUser | null>(null);

  const selectedUser = useMemo(() => {
    return users.find((user) => user.id === selectedUserID) ?? null;
  }, [users, selectedUserID]);

  const currentStatusLabel = useMemo(() => {
    if (!selectedUser) {
      return "未選択";
    }
    return selectedUser.status === "ACTIVE" ? "有効" : "無効";
  }, [selectedUser]);
  const currentStatus = selectedUser?.status ?? null;

  const canSubmit = useMemo(() => {
    return isSessionReady && adminToken.trim().length > 0 && selectedUserID.trim().length > 0 && !isSubmitting && !isFetchingUsers;
  }, [isSessionReady, adminToken, selectedUserID, isSubmitting, isFetchingUsers]);

  useEffect(() => {
    if (selectedUser) {
      setStatus(selectedUser.status);
    }
  }, [selectedUser]);

  async function fetchUsers(token: string, currentAdminUserID: string) {
    if (token.trim().length === 0) {
      return;
    }

    setIsFetchingUsers(true);
    setErrorMessage("");
    setUpdatedUser(null);

    try {
      const response = await fetch("/api/admin/users", {
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        if (response.status === 401) {
          clearAuthSession();
        }
        const errorBody = (await response.json().catch(() => ({}))) as ErrorResponse;
        setErrorMessage(formatErrorMessage(response.status, errorBody.error ?? ""));
        return;
      }

      const result = (await response.json()) as AdminUsersResponse;
      const manageableUsers = result.users.filter((user) => user.id !== currentAdminUserID);
      setUsers(manageableUsers);

      if (manageableUsers.length === 0) {
        setSelectedUserID("");
        setErrorMessage("変更可能なユーザーが見つかりませんでした。");
        return;
      }

      const hasCurrent = manageableUsers.some((user) => user.id === selectedUserID);
      if (!hasCurrent) {
        setSelectedUserID(manageableUsers[0].id);
      }
    } catch {
      setErrorMessage("ユーザー一覧の取得中にネットワークエラーが発生しました。");
    } finally {
      setIsFetchingUsers(false);
    }
  }

  useEffect(() => {
    const session = loadAuthSession();
    if (!session) {
      setErrorMessage("セッション情報を確認できませんでした。再ログインしてください。");
      setIsSessionReady(true);
      return;
    }
    if (isSessionExpired(session.expiresAt)) {
      clearAuthSession();
      setErrorMessage("セッションの有効期限が切れました。再ログインしてください。");
      setIsSessionReady(true);
      return;
    }

    const currentAdminUserID = extractUserIDFromToken(session.token);
    setAdminUserID(currentAdminUserID);
    setAdminToken(session.token);
    setIsSessionReady(true);
    void fetchUsers(session.token, currentAdminUserID);
  }, []);

  // セッション判定が終わるまで管理者画面本体を描画しない。
  // これにより、一般利用者の直リンク時に一瞬だけ管理者UIが見える現象を防ぐ。
  if (!isSessionReady) {
    return null;
  }

  // 保存済みの管理者セッションで status 更新 API を実行する。
  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSubmit) {
      return;
    }

    setIsSubmitting(true);
    setErrorMessage("");
    setUpdatedUser(null);

    try {
      const targetUserID = selectedUserID.trim();
      if (targetUserID === adminUserID) {
        setErrorMessage("自分自身の利用状態は変更できません。");
        return;
      }

      const response = await fetch(`/api/admin/users/${encodeURIComponent(targetUserID)}/status`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${adminToken.trim()}`,
        },
        body: JSON.stringify({ status }),
      });

      if (!response.ok) {
        if (response.status === 401) {
          clearAuthSession();
        }
        const errorBody = (await response.json().catch(() => ({}))) as ErrorResponse;
        setErrorMessage(formatErrorMessage(response.status, errorBody.error ?? ""));
        return;
      }

      const result = (await response.json()) as UpdateUserStatusResponse;
      const updatedTargetUser = users.find((user) => user.id === result.user.id);
      setUpdatedUser({
        ...result.user,
        employee_id: updatedTargetUser?.employee_id ?? "-",
        email: updatedTargetUser?.email ?? "-",
      });
      setUsers((prevUsers) =>
        prevUsers.map((user) => (user.id === result.user.id ? { ...user, status: result.user.status } : user)),
      );
    } catch {
      setErrorMessage("ネットワークエラーが発生しました。接続状態を確認してください。");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className={`${styles.shell} min-h-[calc(100vh-7.5rem)] py-4`}>
      <main className={`${styles.container} flex flex-col gap-4`}>
        <h1 className={styles.heading}>利用状態の変更</h1>
        <p className={styles.lead}>
          管理者の利用者管理として、対象ユーザーを「有効」または「無効」に変更できます。
        </p>

        <section
          className={`${styles.card} rounded-[1rem] border border-slate-200 bg-white/96 px-6 pt-4 pb-6 shadow-[0_20px_38px_rgba(15,23,42,0.09)] sm:px-7 sm:pt-5 sm:pb-7`}
          aria-label="ユーザー状態更新フォーム"
        >
          <div className={styles.cardBody}>
            <UserStatusForm
              users={users}
              selectedUserID={selectedUserID}
              currentStatus={currentStatus}
              currentStatusLabel={currentStatusLabel}
              status={status}
              isSubmitting={isSubmitting}
              canSubmit={canSubmit}
              onSelectedUserIDChange={setSelectedUserID}
              onStatusChange={setStatus}
              onSubmit={handleSubmit}
            />

            <p className={styles.notice}>
              この操作は管理者のみ実行できます。ログイン中の管理者権限で更新を行います。
            </p>

            <UserStatusFeedback errorMessage={errorMessage} updatedUser={updatedUser} />
          </div>
        </section>
      </main>
    </div>
  );
}
