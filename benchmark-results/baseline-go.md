# Baseline pós-upgrade (set/2026) — Go

Branch: `feature/upgrade-benchmark` (base: `main`)

## Procedimento
- `docker compose -f docker-compose.benchmark.yml -p bench-go up -d` (postgres 18.6-alpine, nginx 1.30.4-alpine, tabela LOGGED)
- k6 `--summary-trend-stats "avg,min,med,max,p(90),p(95),p(99)"`, settle de 20s entre cenários pesados
- Logs brutos em `benchmark-results/baseline-go-*.log`

## Resultados (todas as suítes verdes, exit=0)

| Cenário | p95 | p99 | RPS |
|---|---|---|---|
| smoke | 3.88ms | 4.61ms | 475 |
| post-heavy | 156.57ms | 343.3ms | 2184 |
| search-heavy | 48.75ms | 49.88ms | 2209 |
| get-by-id-heavy | 1.41ms | 1.93ms | 4673 |
| mixed-rinha-like | 598.93ms | 993.24ms | 515 |

## Contrato
- contract-ko: 8/8 checks OK

## Notas de metodologia
- Scripts k6 normalizados entre os 3 repos (cópias do C++, threshold `http_req_failed{expected_response:true}`) — os antigos tinham threshold que contava KO intencional como falha (exits 99 falsos)
- search-heavy p95=5s na 1ª corrida era ruído de checkpoint/vacuum do DB pós-post-heavy → settle de 20s no runbook

## Achados para otimização (Fase 3)
1. `post-heavy` p99 343ms vs 152ms do Rust — lado Go (json.Marshal? pgx?) a investigar
2. `mixed-rinha-like` p99 ~1s, puxado por search + escritas
