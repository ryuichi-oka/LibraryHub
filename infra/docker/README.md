# infra/docker

Docker Compose 関連ファイルを配置するディレクトリ。
現在の起動定義はルートの [docker-compose.yml](../../docker-compose.yml) を使用する。

- [api/Dockerfile](./api/Dockerfile): Go API コンテナのビルド定義
- [web/Dockerfile](./web/Dockerfile): Next.js Web コンテナのビルド定義
