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

## Benchmark com Gatling (Linux recomendado)

```bash
# Pré-requisitos: Java 17+, git, curl
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

## Benchmark apontando para outra máquina

Edite o `baseUrl` no arquivo de simulação do Gatling:

```scala
// /tmp/rinha/stress-test/user-files/simulations/rinhabackend/RinhaBackendSimulation.scala
.baseUrl("http://<IP-DA-MAQUINA>:9999")
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
Gatling → LB (Pingora Rust :9999) → api1 (Go :8080)
                                  → api2 (Go :8081)
                                  → db (Postgres :5432)
```

- O LB usa a imagem `backend-cockfighting-q3-2023-lb` com tag SHA fixa em `docker-compose.benchmark.yml`
- API images pinned by SHA (evita regressão)

## Troubleshooting

| Erro | Causa provável |
|---|---|
| `Cannot assign requested address` | Portas efêmeras esgotadas no macOS Docker Desktop |
| `Premature close` | Proxy HTTP sem keep-alive ou timeout baixo |
| `Connection refused` no `/contagem-pessoas` | API/LB caiu durante o stress |
| `no rows in result set` / 422 Conflict | Apelido duplicado (esperado) |
