# backend-cockfighting-q3-2023-go

API em Go para a Rinha de Backend 2023 Q3.

Stack: `net/http` (Go 1.22+ mux) + `pgx/v5` + `uuid`

---

## Quick start

```bash
# Teste local (API + DB)
docker compose -f docker-compose.test.yml up -d
curl http://localhost:8080/health-check

# Parar
docker compose -f docker-compose.test.yml down -v
```

## Benchmark local

```bash
# Stack completa com 2 APIs + LB + DB
docker compose -f docker-compose.benchmark.yml up -d

# Aguardar health
for i in $(seq 1 20); do
  if curl -s -o /dev/null -w "%{http_code}" http://localhost:9999/health-check | grep -q 200; then
    echo "ready"; break
  fi; sleep 2
done

# Testar endpoints
curl -s "http://localhost:9999/pessoas?t=node"
curl -s "http://localhost:9999/contagem-pessoas"
curl -s -o /dev/null -w "%{http_code}\n" "http://localhost:9999/pessoas"

# Parar
docker compose -f docker-compose.benchmark.yml down -v
```

## Benchmark com k6

### Instalar k6

```bash
# Ubuntu/Debian
sudo apt-get install -y gnupg
curl -fsSL https://dl.k6.io/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/k6.gpg
echo "deb [signed-by=/usr/share/keyrings/k6.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install -y k6

# macOS
brew install k6
```

### Scripts disponíveis

| Script | Objetivo | Uso recomendado |
|---|---|---|
| `benchmarks/smoke.js` | Valida contrato rápido | CI/PR |
| `benchmarks/post-heavy.js` | Escrita pesada | Benchmark local |
| `benchmarks/search-heavy.js` | Carga de busca | Diagnóstico de índice |
| `benchmarks/get-by-id-heavy.js` | Lookup por UUID | Medir leitura simples |
| `benchmarks/mixed-rinha-like.js` | Mix próximo da rinha | Benchmark geral |
| `benchmarks/contract-ko.js` | Status code exatos | Validar contrato HTTP |

### Rodar local

```bash
# Por padrão aponta para localhost:9999 (stack completa)
k6 run benchmarks/smoke.js

# Ou apontar para API direta
BASE_URL=http://localhost:8080 k6 run benchmarks/smoke.js
```

## Benchmark com Gatling

Pré-requisitos: Linux (recomendado), Java 17+, git, curl.

```bash
git clone https://github.com/zanfranceschi/rinha-de-backend-2023-q3 /tmp/rinha
cd /tmp/rinha
curl -sL -o gatling.zip https://repo1.maven.org/maven2/io/gatling/highcharts/gatling-charts-highcharts-bundle/3.9.5/gatling-charts-highcharts-bundle-3.9.5-bundle.zip
unzip -q gatling.zip
cd gatling-charts-highcharts-bundle-3.9.5

# Rodar contra a stack local
./bin/gatling.sh -rm local -s RinhaBackendSimulation \
  -rd "benchmark-go" \
  -rf ./results \
  -sf /tmp/rinha/stress-test/user-files/simulations \
  -rsf /tmp/rinha/stress-test/user-files/resources
```

> ⚠️ No macOS/Docker Desktop, o teste pode falhar por esgotamento de portas efêmeras (`Cannot assign requested address`). Prefira Linux nativo ou GitHub Actions.

## Benchmark remoto

### k6
```bash
BASE_URL=http://<IP>:9999 k6 run benchmarks/mixed-rinha-like.js
```

### Gatling
Editar `baseUrl` no arquivo de simulação:
```scala
// stress-test/user-files/simulations/rinhabackend/RinhaBackendSimulation.scala
.baseUrl("http://<IP>:9999")
```

## Variáveis de ambiente

| Variável | Padrão | Descrição |
|---|---|---|
| `PORT` | `8080` | Porta da API |
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_NAME` | `fight` | Nome do banco |
| `DB_USER` | `postgres` | Usuário do banco |
| `DB_PASSWORD` | `fight` | Senha do banco |

## Arquitetura do benchmark

```
Gatling/k6 → nginx (:9999) → api1 (Go :8080)
                            → api2 (Go :8081)
                            → db (Postgres :5432)
```

- A imagem do nginx é pública (`nginx:1.27.4-alpine`)
- A imagem da API é `ghcr.io/lucaswilliameufrasio/backend-cockfighting-q3-2023-go:latest`
- O CI faz push de `latest` + SHA a cada push em `main`

## Troubleshooting

| Erro | Causa provável |
|---|---|
| `Cannot assign requested address` | Portas efêmeras esgotadas no macOS Docker Desktop |
| `Premature close` | Proxy HTTP sem keep-alive ou timeout baixo |
| `Connection refused` no `/contagem-pessoas` | API/LB caiu durante o stress |
| `no rows in result set` / 422 Conflict | Apelido duplicado (esperado) |
| `manifest unknown` no pull da imagem | SHA da imagem não existe no registry. Use `latest` ou uma SHA de um CI que passou. |
