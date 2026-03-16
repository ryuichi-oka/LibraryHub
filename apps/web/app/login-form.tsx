"use client";

import { FormEvent, useMemo, useState } from "react";
import styles from "./login-form.module.css";

type LoginResponse = {
  token_type: string;
  token: string;
  expires_at: string;
  user: {
    id: string;
    employee_id: string;
    email: string;
    role: string;
  };
};

type ErrorResponse = {
  error?: string;
};

const LOGIN_ENDPOINT = "/api/auth/login";

function formatErrorMessage(status: number, fallback: string): string {
  if (status === 401) {
    return "社員ID / メールアドレスまたはパスワードが正しくありません。";
  }
  if (status === 400) {
    return "入力内容を確認して、もう一度お試しください。";
  }
  return fallback || "ログインに失敗しました。時間をおいて再試行してください。";
}

// LoginForm は identifier + password でのログインを受け付ける。
export default function LoginForm() {
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [loginResult, setLoginResult] = useState<LoginResponse | null>(null);

  const canSubmit = useMemo(() => {
    return identifier.trim().length > 0 && password.length > 0 && !isSubmitting;
  }, [identifier, password, isSubmitting]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!canSubmit) {
      return;
    }

    setIsSubmitting(true);
    setErrorMessage("");
    setLoginResult(null);

    try {
      const response = await fetch(LOGIN_ENDPOINT, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          identifier: identifier.trim(),
          password,
        }),
      });

      if (!response.ok) {
        const errorBody = (await response.json().catch(() => ({}))) as ErrorResponse;
        setErrorMessage(formatErrorMessage(response.status, errorBody.error ?? ""));
        return;
      }

      const data = (await response.json()) as LoginResponse;
      setLoginResult(data);
      setPassword("");
    } catch {
      setErrorMessage("ネットワークエラーが発生しました。接続状態を確認してください。");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className={`${styles.shell} min-h-screen bg-slate-50 px-5 py-8 sm:px-8`}>
      <div className={`${styles.bgOrb} ${styles.bgOrbLeft}`} aria-hidden="true" />
      <div className={`${styles.bgOrb} ${styles.bgOrbRight}`} aria-hidden="true" />

      <main className={`${styles.centerWrap} flex flex-col gap-4`}>
        <header className={styles.brandHeader}>
          <p className={styles.brandEyebrow}>LIBRARY PORTAL</p>
          <h1 className={styles.brandTitle}>LIBRARYHUB</h1>
          <p className={styles.brandSubTitle}>図書管理システム</p>
        </header>

        <section
          className="rounded-[1rem] border border-slate-200 bg-white/96 px-6 pt-4 pb-6 shadow-[0_20px_38px_rgba(15,23,42,0.09)] sm:px-7 sm:pt-5 sm:pb-7"
          aria-label="ログインフォーム"
        >
          <div>
            <h2 className={styles.formTitle}>ログイン</h2>
            <p className={styles.formLead}>メールアドレスまたは社員IDを入力してください。</p>
          </div>

          <form className="mt-6 space-y-4" onSubmit={handleSubmit} noValidate>
            <label className="block" htmlFor="identifier">
              <span className={styles.fieldLabel}>メールアドレス / 社員ID</span>
              <input
                id="identifier"
                name="identifier"
                type="text"
                autoComplete="username"
                placeholder="例: user@example.com または E001"
                value={identifier}
                onChange={(event) => setIdentifier(event.target.value)}
                disabled={isSubmitting}
                className={`${styles.textInput} mt-2 w-full`}
                required
              />
            </label>

            <label className="block" htmlFor="password">
              <span className={styles.fieldLabel}>パスワード</span>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                placeholder="パスワードを入力"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                disabled={isSubmitting}
                className={`${styles.textInput} mt-2 w-full`}
                required
              />
            </label>

            {errorMessage ? (
              <p className="rounded-[0.5rem] border border-rose-300 bg-rose-50 px-3 py-2 text-sm leading-6 text-rose-800" role="alert">
                {errorMessage}
              </p>
            ) : null}

            {loginResult ? (
              <div
                className="rounded-[0.5rem] border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm leading-6 text-emerald-800"
                role="status"
                aria-live="polite"
              >
                <p>ログインに成功しました。</p>
                <p className="mt-1">ユーザー: {loginResult.user.employee_id} / 権限: {loginResult.user.role}</p>
              </div>
            ) : null}

            <button
              type="submit"
              disabled={!canSubmit}
              className={`${styles.submitButton} mt-1 w-full`}
            >
              <span className={styles.submitButtonLabel}>{isSubmitting ? "ログイン中..." : "ログイン"}</span>
            </button>
          </form>
        </section>
      </main>
    </div>
  );
}
