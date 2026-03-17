import styles from "./UserStatusScreen.module.css";
import { UpdatedUser } from "./types";

function toStatusLabel(status: UpdatedUser["status"]): string {
  if (status === "ACTIVE") {
    return "有効";
  }
  return "無効";
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
          <p>利用者ID: {updatedUser.id}</p>
          <p>現在の状態: {toStatusLabel(updatedUser.status)}</p>
          <p>更新日時: {updatedUser.updated_at}</p>
        </div>
      ) : null}
    </>
  );
}
