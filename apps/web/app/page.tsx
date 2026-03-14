async function getApiHealth(): Promise<string> {
  try {
    const response = await fetch("http://api:8080/healthz", {
      cache: "no-store",
    });
    if (!response.ok) {
      return "API unreachable";
    }
    const data = (await response.json()) as { status?: string };
    return data.status ?? "unknown";
  } catch {
    return "API unreachable";
  }
}

export default async function HomePage() {
  const apiStatus = await getApiHealth();

  return (
    <main className="container">
      <h1>LibraryHub</h1>
      <p>Docker 開発環境の初期画面です。</p>
      <p>
        API health: <strong>{apiStatus}</strong>
      </p>
    </main>
  );
}
