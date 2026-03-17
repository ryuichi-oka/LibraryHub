import styles from "./LoginScreen.module.css";

type FormFieldProps = {
  id: string;
  label: string;
  type: "text" | "password";
  autoComplete: string;
  placeholder: string;
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
};

export default function FormField({
  id,
  label,
  type,
  autoComplete,
  placeholder,
  value,
  disabled,
  onChange,
}: FormFieldProps) {
  return (
    <label className="block" htmlFor={id}>
      <span className={styles.fieldLabel}>{label}</span>
      <input
        id={id}
        name={id}
        type={type}
        autoComplete={autoComplete}
        placeholder={placeholder}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
        className={`${styles.textInput} mt-2 w-full`}
        required
      />
    </label>
  );
}
