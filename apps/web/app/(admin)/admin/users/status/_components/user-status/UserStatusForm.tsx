import styles from "./UserStatusScreen.module.css";
import { AdminUser, UserStatus } from "./types";

const USER_STATUS_OPTIONS: Array<{ value: UserStatus; label: string }> = [
  { value: "ACTIVE", label: "有効" },
  { value: "INACTIVE", label: "無効" },
];

type UserStatusFormProps = {
  users: AdminUser[];
  selectedUserID: string;
  currentStatusLabel: string;
  status: UserStatus;
  isSubmitting: boolean;
  canSubmit: boolean;
  onSelectedUserIDChange: (value: string) => void;
  onStatusChange: (status: UserStatus) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
};

export default function UserStatusForm({
  users,
  selectedUserID,
  currentStatusLabel,
  status,
  isSubmitting,
  canSubmit,
  onSelectedUserIDChange,
  onStatusChange,
  onSubmit,
}: UserStatusFormProps) {
  return (
    <form onSubmit={onSubmit} className={styles.form} noValidate>
      <label htmlFor="target-user" className={styles.fieldBlock}>
        <span className={styles.label}>ユーザー</span>
        <select
          id="target-user"
          name="target-user"
          value={selectedUserID}
          onChange={(event) => onSelectedUserIDChange(event.target.value)}
          className={styles.textInput}
          disabled={users.length === 0 || isSubmitting}
          required
        >
          <option value="">ユーザーを選択してください</option>
          {users.map((user) => (
            <option key={user.id} value={user.id}>
              {user.employee_id} / {user.email}
            </option>
          ))}
        </select>
      </label>

      <p className={styles.currentStatus}>現在の状態: {currentStatusLabel}</p>

      <div className={styles.fieldBlock}>
        <span className={styles.label}>変更後の状態</span>
        <div className={styles.statusGroup} role="group" aria-label="変更後の状態の選択">
          {USER_STATUS_OPTIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              className={`${styles.statusButton} ${status === option.value ? styles.statusButtonSelected : ""}`}
              onClick={() => onStatusChange(option.value)}
              disabled={isSubmitting}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <button type="submit" disabled={!canSubmit} className={styles.submitButton}>
        {isSubmitting ? "更新中..." : "利用状態を変更する"}
      </button>
    </form>
  );
}
