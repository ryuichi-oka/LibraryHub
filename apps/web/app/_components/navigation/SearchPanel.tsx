"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

import styles from "./AppShell.module.css";

const SEARCH_CATEGORY_OPTIONS = [
  "技術書",
  "デザイン",
  "ビジネス",
  "マネジメント",
  "小説",
  "語学",
  "雑誌",
  "自己啓発",
  "資格",
  "その他",
];

export default function SearchPanel() {
  const router = useRouter();
  const [searchKeyword, setSearchKeyword] = useState("");
  const [selectedCategories, setSelectedCategories] = useState<string[]>([]);

  function handleSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const query = new URLSearchParams();
    if (searchKeyword.trim().length > 0) {
      query.set("q", searchKeyword.trim());
    }
    selectedCategories.forEach((category) => query.append("category", category));

    const queryString = query.toString();
    router.push(queryString.length > 0 ? `/home?${queryString}` : "/home");
  }

  function handleCategoryChange(category: string, checked: boolean) {
    setSelectedCategories((prev) => {
      if (checked) {
        if (prev.includes(category)) {
          return prev;
        }
        return [...prev, category];
      }
      return prev.filter((item) => item !== category);
    });
  }

  return (
    <aside className={`w-full max-w-[20rem] shrink-0 ${styles.searchAside}`}>
      <section className={styles.searchPanel} aria-label="図書検索">
        <h2 className={styles.searchTitle}>検索</h2>

        <form className={styles.searchForm} onSubmit={handleSearchSubmit}>
          <label className={styles.searchLabel} htmlFor="search-keyword">
            書名・著者・ISBN
          </label>
          <input
            id="search-keyword"
            className={styles.searchInput}
            type="text"
            value={searchKeyword}
            onChange={(event) => setSearchKeyword(event.target.value)}
            placeholder="例: リーダブルコード"
          />

          <p className={styles.searchLabel}>カテゴリ（複数選択）</p>
          <div className={styles.categoryList} role="group" aria-label="カテゴリ選択">
            {SEARCH_CATEGORY_OPTIONS.map((category) => {
              const checked = selectedCategories.includes(category);
              return (
                <label key={category} className={styles.categoryItem}>
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={(event) => handleCategoryChange(category, event.target.checked)}
                  />
                  <span>{category}</span>
                </label>
              );
            })}
          </div>

          <button type="submit" className={styles.searchButton}>
            検索する
          </button>
        </form>
      </section>
    </aside>
  );
}
