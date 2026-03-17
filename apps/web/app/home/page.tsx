export default function HomePage() {
  return (
    <main className="min-h-screen bg-slate-50 px-5 py-8 sm:px-8">
      <section className="mx-auto max-w-3xl rounded-[1rem] border border-slate-200 bg-white/96 px-6 pt-4 pb-6 shadow-[0_20px_38px_rgba(15,23,42,0.09)] sm:px-7 sm:pt-5 sm:pb-7">
        <h1 className="text-[1.45rem] font-bold tracking-[0.02em] text-slate-700">トップ画面</h1>
        <p className="mt-3 text-[0.98rem] leading-7 text-slate-600">
          新着一覧とおすすめ表示は後続タスクで実装予定です。現在はログイン後の遷移先としてこの画面を利用します。
        </p>
      </section>
    </main>
  );
}
