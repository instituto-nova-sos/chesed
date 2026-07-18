# Chesed — Relatório de Validação

**Autor:** Pedro Barbosa (com apoio de IA / Claude Code)
**Data:** 2026-07-14
**Escopo:** Validação técnica da "primeira versão" do software, execução local ponta a ponta, revisão de segurança/token e riscos de dev sênior.
**Commit avaliado:** `origin/main` (PR #42 — *volunteer agreement signature*)

---

## ⚠️ Achado nº 1 — Cópia local desatualizada (ler antes da call)

A cópia local do repositório estava **53 commits atrasada**: o branch `main` no disco apontava para `b9b00d0 "ai factory"` — a fase **só de documentação**, antes de qualquer código. Por isso o repositório parecia "vazio de código".

O software anunciado ("finalizamos a primeira versão") **existe e é real** — está no `origin/main` do GitHub, 53 commits à frente, com 12 sprints (até o PR #42). Foi feito `git pull` (fast-forward seguro, nada local perdido) e a máquina está sincronizada.

**Ações de processo:**
- Todo o time deve dar `git pull` antes de validar — senão validam a versão errada.
- Confirmar que a `main` do GitHub é a **fonte da verdade** (a versão do disco era uma linha paralela só de docs).

---

## Parte 1 — Estado do repositório

| Item | Situação |
|---|---|
| Backend Go | **187** arquivos `.go` + **84** de teste; camadas handler→service→repository→domain |
| Frontend React/TS | **229** arquivos `.ts/.tsx` + **79** de teste |
| Migrations SQL | **33** (todas com `.up`/`.down`) |
| CI/CD | 4 workflows: `backend`, `frontend`, `security`, `deploy` |
| Bibliotecas | go-oidc, chi, pgx, httprate (rate limit), golang-migrate, validator — 100% alinhado ao `CLAUDE.md` |

**Conclusão:** não é protótipo — é uma aplicação madura. Cobre LGPD, multi-região e assinatura de termo de voluntário.

---

## Parte 2 — Execução local (validada)

Stack completo subido com `docker compose up -d --build` (6 serviços):

| Verificação | Resultado |
|---|---|
| Postgres, Keycloak 26, MinIO, Mailpit | **healthy** |
| 33 migrations aplicadas | **OK, sem erro** |
| Realm Keycloak + 5 usuários de teste | **OK** |
| `GET /api/v1/health` | **200** — `{"database":"ok","status":"ok"}` |
| Frontend em `:5173` | **200** (serve HTML) |
| Endpoint protegido **sem** token | **401** (correto) |
| Testes de segurança (middleware/auth/domain) | **PASS** |
| Typecheck do frontend (`tsc --noEmit`) | **limpo, 0 erros** |

### Como rodar
```bash
git pull
docker compose up -d --build
cd backend && make migrate-up      # se faltar o CLI 'migrate' no host, rodar dentro do container
./keycloak/init-realm.sh
# Frontend: http://localhost:5173
```

> O host não tinha o CLI `golang-migrate`; as migrations foram rodadas por dentro do container `api`. Recomenda-se instalar (`brew install golang-migrate` ou equivalente) para não travar o time no passo 3.

### URLs de acesso (ambiente local)

| Serviço | URL | Credenciais |
|---|---|---|
| **Frontend (sistema)** | http://localhost:5173 | login abaixo |
| API (health) | http://localhost:8080/api/v1/health | — |
| Keycloak Admin | http://localhost:8180/admin | `admin` / `admin` |
| Mailpit (e-mails) | http://localhost:8025 | — |
| MinIO Console (S3) | http://localhost:9001 | `chesed` / `chesed-dev-secret` |

### Usuários de teste (senha `Test1234!`)

| Usuário | Papel |
|---|---|
| `volunteer@chesed.test` | VOLUNTEER |
| `secretary@chesed.test` | SECRETARY |
| `professional@chesed.test` | PROFESSIONAL |
| `coordinator@chesed.test` | COORDINATOR |
| `admin@chesed.test` | ADMIN |

---

## Parte 3 — 🔐 Segurança de token ("safe token")

O tratamento de token está **bem feito** — ponto forte do projeto:

- ✅ **PKCE S256** no login (authorization-code flow)
- ✅ Access token **só em memória** (Zustand), nunca em `localStorage` → sem roubo por XSS persistente
- ✅ **Direct Access Grants DESABILITADO** no client `chesed-pwa` (confirmado via API do Keycloak: grant de senha retorna `unauthorized_client`). Token só é emitido pelo fluxo de browser.
- ✅ Refresh com **rotação**; `getToken()` renova antes de cada chamada
- ✅ Backend valida assinatura **RS256 via JWKS** (go-oidc) com checagem de **audience** (`ClientID`)
- ✅ Isolamento por campus via **Row-Level Security do Postgres** (`SET LOCAL app.current_campus`) — protege leitura **e** escrita no nível do banco

### ❗ Ajustes obrigatórios antes de produção (o modo dev afrouxa)
1. `OIDC_SKIP_ISSUER_CHECK=true` → **desligar** em produção
2. **MFA desabilitado** e e-mail não verificado → habilitar (`verifyEmail=true`, MFA condicional)
3. Senhas/segredos triviais de dev (`admin/admin`, `Test1234!`, secrets no compose) → substituir por secrets reais

---

## Parte 4 — Bugs e riscos (visão de dev sênior)

> **Nota importante:** os documentos de design em `docs/` estão **desatualizados** e descrevem falhas graves (escalonamento de privilégio no RBAC, bug de SQL na migration, token em cookie) que **o código já corrigiu**. Verificado no código:
> - RBAC usa allow-list explícita (`HasRole(roles...)`), não a "escada" quebrada dos docs → **sem escalonamento de privilégio**
> - Migration usa `UNIQUE NULLS NOT DISTINCT` (válido), não o `UNIQUE ... WHERE` inválido → **compila e roda**

Riscos **reais** remanescentes:

| # | Risco | Severidade |
|---|---|---|
| R1 | **Docs de design desatualizados** contradizem o código. Novos integrantes vão se confundir. Reconciliar `docs/` com a realidade. | Alto (processo) |
| R2 | **Confusão de branch** (`main` limpa vs. `main` com código). Definir fonte da verdade e fluxo de merge. | Alto (processo) |
| R3 | **Hardening de produção** pendente (issuer check, MFA, secrets, TLS Cloudflare↔origem). Ver Parte 3. | Alto (segurança) |
| R4 | **Offline sync** — parte mais arriscada de PWA offline-first (ordem de FK, duplicatas entre 2 celulares no mesmo evento, resolução de conflito). Já há índice único de `sync_id` e um "drainer", mas requer **teste de campo dedicado**. | Médio — testar |
| R5 | **Deploy free-tier** (Render) que "dorme" após 15min conflita com meta de 3s. Alinhar com o Bruno no deploy. | Médio |
| R6 | Sem meta de disponibilidade (uptime); retenção/erasure LGPD parcial. | Baixo-médio |

---

## Parte 5 — Pontos para o grupo / call

Além do que já foi listado (testar, validar com a Carla, revisão de UI/UX, definir deploy, formar time):

1. **Sincronizar o repositório primeiro** — todos na mesma versão (`main` do GitHub).
2. **Reconciliar documentação × código** antes de trazer gente nova (ex.: SOS Capacita), senão programam contra specs erradas.
3. **Checklist de hardening de produção** como pré-requisito de deploy (os 3 itens da Parte 3).
4. **Sessão de teste de campo do modo offline** com 2+ celulares — onde dados de beneficiários podem se perder.
5. Confirmar a **hospedagem** (free-tier atual não atende performance) — pauta do Bruno.

**Resumo de uma linha:** *"A primeira versão está pronta, roda localmente e a segurança/token está sólida. Antes do uso real faltam: hardening de produção, reconciliar documentação com código, e um teste dedicado do modo offline."*

---

## Anexo — Evidências de execução

```
GET /api/v1/health            -> 200  {"database":"ok","status":"ok"}
GET /  (frontend :5173)       -> 200  (HTML)
GET /api/v1/persons (s/ token)-> 401  {"error":"unauthorized"}
chesed-pwa directAccessGrants -> false  (password grant bloqueado)
go test ./internal/middleware/... ./internal/auth/... ./internal/domain/... -> ok
tsc --noEmit                  -> 0 erros
migrations                    -> 33/33 aplicadas
```
