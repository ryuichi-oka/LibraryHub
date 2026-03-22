import styles from "./UserStatusScreen.module.css";
import { UpdatedUser } from "./types";

function toStatusLabel(status: UpdatedUser["status"]): string {
  if (status === "ACTIVE") {
    return "有効";
  }
  return "無効";
}

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");
  return `${year}/${month}/${day} ${hours}:${minutes}:${seconds}`;
}

type UserStatusFeedbackProps = {
  errorMessage: string;
  updatedUser: UpdatedUser | null;
};

export default function UserStatusFeedback({ errorMessage, updatedUser }: UserStatusFeedbackProps) {
  return (
    <>
      {errorMessage ? (
        <p role="alert" className={styles.error}>
          {errorMessage}
        </p>
      ) : null}

      {updatedUser ? (
        <div role="status" aria-live="polite" className={styles.success}>
          <p>利用状態を変更しました。</p>
          <p>ユーザー名: {updatedUser.employee_id}</p>
          <p>メールアドレス: {updatedUser.email}</p>
          <p>現在の状態: {toStatusLabel(updatedUser.status)}</p>
          <p>更新日時: {formatDateTime(updatedUser.updated_at)}</p>
        </div>
      ) : null}
    </>
  );
}
