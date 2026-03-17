"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { clearAuthSession, isSessionExpired, loadAuthSession } from "../../../../../../_lib/authSession";
import styles from "./UserStatusScreen.module.css";
import UserStatusFeedback from "./UserStatusFeedback";
import UserStatusForm from "./UserStatusForm";
import { AdminUser, AdminUsersResponse, ErrorResponse, UpdateUserStatusResponse, UserStatus } from "./types";

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
  return fallback || "ステータス更新に失敗しました。時間をおいて再試行してください。";
}

export default function UserStatusScreen() {
  const [adminToken, setAdminToken] = useState("");
  const [isSessionReady, setIsSessionReady] = useState(false);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [selectedUserID, setSelectedUserID] = useState("");
  const [status, setStatus] = useState<UserStatus>("INACTIVE");
  const [isFetchingUsers, setIsFetchingUsers] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [updatedUser, setUpdatedUser] = useState<UpdateUserStatusResponse["user"] | null>(null);

  const selectedUser = useMemo(() => {
    return users.find((user) => user.id === selectedUserID) ?? null;
  }, [users, selectedUserID]);

  const currentStatusLabel = useMemo(() => {
    if (!selectedUser) {
      return "未選択";
    }
    return selectedUser.status === "ACTIVE" ? "有効" : "無効";
  }, [selectedUser]);

  const canSubmit = useMemo(() => {
    return isSessionReady && adminToken.trim().length > 0 && selectedUserID.trim().length > 0 && !isSubmitting && !isFetchingUsers;
  }, [isSessionReady, adminToken, selectedUserID, isSubmitting, isFetchingUsers]);

  useEffect(() => {
    if (selectedUser) {
      setStatus(selectedUser.status);
    }
  }, [selectedUser]);

  async function fetchUsers(token: string) {
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
      setUsers(result.users);

      if (result.users.length === 0) {
        setSelectedUserID("");
        setErrorMessage("選択可能なユーザーが見つかりませんでした。");
        return;
      }

      const hasCurrent = result.users.some((user) => user.id === selectedUserID);
      if (!hasCurrent) {
        setSelectedUserID(result.users[0].id);
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
      setErrorMessage("ログイン情報が見つかりません。ログイン画面からやり直してください。");
      setIsSessionReady(true);
      return;
    }
    if (isSessionExpired(session.expiresAt)) {
      clearAuthSession();
      setErrorMessage("ログインの有効期限が切れました。再ログインしてください。");
      setIsSessionReady(true);
      return;
    }
    if (session.role !== "ADMIN") {
      setErrorMessage("この画面は管理者のみ利用できます。");
      setIsSessionReady(true);
      return;
    }

    setAdminToken(session.token);
    setIsSessionReady(true);
    void fetchUsers(session.token);
  }, []);

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
      setUpdatedUser(result.user);
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
    <div className={`${styles.shell} min-h-screen bg-slate-50 px-5 py-8 sm:px-8`}>
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
