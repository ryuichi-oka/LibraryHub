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

  const parts = new Intl.DateTimeFormat("ja-JP", {
    timeZone: "Asia/Tokyo",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hourCycle: "h23",
  }).formatToParts(date);

  const partMap = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  const year = partMap.year ?? "";
  const month = partMap.month ?? "";
  const day = partMap.day ?? "";
  const hours = partMap.hour ?? "";
  const minutes = partMap.minute ?? "";
  const seconds = partMap.second ?? "";
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
