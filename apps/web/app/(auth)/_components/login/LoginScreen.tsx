"use client";

import { FormEvent, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { saveAuthSession } from "../../../_lib/authSession";
import { getDefaultPathByRole } from "../../../_lib/authorization";
import BrandHeader from "./BrandHeader";
import LoginFormCard from "./LoginFormCard";
import styles from "./LoginScreen.module.css";

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

export default function LoginScreen() {
  const router = useRouter();
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
      saveAuthSession({
        token: data.token,
        role: data.user.role,
        expiresAt: data.expires_at,
      });
      setLoginResult(data);
      setPassword("");
      router.push(getDefaultPathByRole(data.user.role));
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
        <BrandHeader />
        <LoginFormCard
          identifier={identifier}
          password={password}
          isSubmitting={isSubmitting}
          canSubmit={canSubmit}
          errorMessage={errorMessage}
          loginResult={loginResult}
          onIdentifierChange={setIdentifier}
          onPasswordChange={setPassword}
          onSubmit={handleSubmit}
        />
      </main>
    </div>
  );
}
