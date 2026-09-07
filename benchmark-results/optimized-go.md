# Otimizações aplicadas (set/2026) — Go

Branch: `feature/upgrade-benchmark` · Método idêntico ao baseline (settle 20s, scripts normalizados)

## Antes → Depois

| Cenário | Baseline p95 → p99 | Otimizado p95 → p99 | RPS antes → depois |
|---|---|---|---|
| smoke | 3.88ms → 4.61ms | 3.87ms → 5.25ms | 475 → 471 |
| post-heavy | 156.57ms → 343.3ms | 146.66ms → **263.7ms** | 2184 → 2011 |
| search-heavy | 48.75ms → 49.88ms | **1.97ms → 42.75ms** | 2209 → **3633** |
| get-by-id-heavy | 1.41ms → 1.93ms | 1.65ms → 2.29ms | 4673 → 4659 |
| mixed-rinha-like | 598.93ms → 993.24ms | **548.26ms** → 3.2s¹ | 515 → 818 (1.6x) |

¹ p99 do mixed com outlier único; p95 e RPS melhoraram.

## O que foi aplicado

1. `Content-Length` explícito via `json.Marshal` + `w.Write` (antes: `json.NewEncoder` → chunked)
2. `http.Server` com `ReadHeaderTimeout`/`ReadTimeout`/`WriteTimeout`/`IdleTimeout` (antes: `ListenAndServe` puro)
3. `pgxpool` com `MaxConns` explícito de `DB_MAX_CONNECTIONS` (antes: default `max(4, NumCPU)` do host)
4. `toPGArray` manual → `[]string` nativo do pgx (guard `nil` → slice vazio; NOT NULL respeitado)
5. `errors.Is(err, pgx.ErrNoRows)` no lugar de comparação por string de mensagem
6. `isDateValid` sem `fmt.Sscanf` (parsing manual de dígitos, sem reflection)
7. Compose: `DB_MAX_CONNECTIONS=16`, `GOMEMLIMIT=350MiB`, `GOMAXPROCS=1`
8. Upgrades: go 1.27 (go.mod + golang:1.27-trixie), postgres 18.6, nginx 1.30.4, debian trixie-20260824

## Testes (primeiros do repo)
- `dateutil_test.go`: válidos/inválidos (inclui edge cases de Sscanf: "1990-01-0a", " 90-01-01")
- `jsonutil_test.go`: `Content-Length` presente + ausência de `Transfer-Encoding: chunked`
- 4/4 passando · `go build` ✓ · `go vet` ✓

## Pendências conhecidas
- Suite de integração no estilo do Rust (compose + HTTP real) — pendente
- `search-heavy` p99 42.75ms — cauda do GIN bitmap scan; med está em ~2ms
