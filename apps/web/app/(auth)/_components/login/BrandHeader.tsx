import styles from "./LoginScreen.module.css";

export default function BrandHeader() {
  return (
    <header className={styles.brandHeader}>
      <p className={styles.brandEyebrow}>LIBRARY PORTAL</p>
      <h1 className={styles.brandTitle}>LIBRARYHUB</h1>
      <p className={styles.brandSubTitle}>図書管理システム</p>
    </header>
  );
}
