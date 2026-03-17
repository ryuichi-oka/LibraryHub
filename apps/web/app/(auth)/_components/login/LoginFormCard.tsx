import FormField from "./FormField";
import styles from "./LoginScreen.module.css";

type LoginResult = {
  user: {
    employee_id: string;
    role: string;
  };
};

type LoginFormCardProps = {
  identifier: string;
  password: string;
  isSubmitting: boolean;
  canSubmit: boolean;
  errorMessage: string;
  loginResult: LoginResult | null;
  onIdentifierChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
};

export default function LoginFormCard({
  identifier,
  password,
  isSubmitting,
  canSubmit,
  errorMessage,
  loginResult,
  onIdentifierChange,
  onPasswordChange,
  onSubmit,
}: LoginFormCardProps) {
  return (
    <section
      className="rounded-[1rem] border border-slate-200 bg-white/96 px-6 pt-4 pb-6 shadow-[0_20px_38px_rgba(15,23,42,0.09)] sm:px-7 sm:pt-5 sm:pb-7"
      aria-label="ログインフォーム"
    >
      <div>
        <h2 className={styles.formTitle}>ログイン</h2>
        <p className={styles.formLead}>メールアドレスまたは社員IDを入力してください。</p>
      </div>

      <form className="mt-6 space-y-4" onSubmit={onSubmit} noValidate>
        <FormField
          id="identifier"
          label="メールアドレス / 社員ID"
          type="text"
          autoComplete="username"
          placeholder="例: user@example.com または E001"
          value={identifier}
          disabled={isSubmitting}
          onChange={onIdentifierChange}
        />

        <FormField
          id="password"
          label="パスワード"
          type="password"
          autoComplete="current-password"
          placeholder="パスワードを入力"
          value={password}
          disabled={isSubmitting}
          onChange={onPasswordChange}
        />

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

        <button type="submit" disabled={!canSubmit} className={`${styles.submitButton} mt-1 w-full`}>
          <span className={styles.submitButtonLabel}>{isSubmitting ? "ログイン中..." : "ログイン"}</span>
        </button>
      </form>
    </section>
  );
}
