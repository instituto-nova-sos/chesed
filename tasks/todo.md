# Plano v3 — Fluxo de entrega autônomo (TDD + E2E real + revisão crítica) + Paralelismo

> Sessão 2026-06-28. v3 = v2 (3 críticos: PM/processo, TDD/E2E, sequenciamento) + **entrega autônoma** + **execução paralela**.
> Decisões do usuário (precedência):
> - E2E = MSW+testcontainers por feature **e** Playwright full-stack nos fluxos críticos.
> - Escopo = refatorar artefatos primeiro, feature depois.
> - Paralelismo = grafo de dependências + subagentes paralelos (execução deste plano também em paralelo).
> - **Entrega autônoma**: todo entregável passa por E2E real **e** por um **agente revisor crítico** (gate bloqueante). O pipeline roda tudo localmente e **para exatamente antes de `git push`/abrir PR/merge** — o PAT do GitHub não tem essa permissão e não será concedido. Limite firme: nunca executar `git push`, `gh pr create`, `gh pr merge`.

---

## Diagnóstico (verificado em código + 3 críticas)

- Backend Phase 1 ~completo (33 arquivos de teste, build OK). Frontend rico mas **4 testes só**, **0 E2E**.
- Próxima fatia: **sync drainer offline do frontend** (Dexie v2, `useOnlineSync`, pull-merge, UI de conflito, TanStack Query).
- Lacunas de gestão: status em fonte única ausente, critérios não-testáveis, sem grafo de dependências, sem DoD, sem sizing, sem rastreabilidade requisito→história, anti-drift ausente (CI pausado).
- Lacunas técnicas: TDD implícito (sem prova de RED); integração para no MSW (sem E2E real); ponto cego de auth Keycloak; assimetria de cobertura frontend.

---

## Princípios da v2 (correções dos críticos)

1. **Escopo mínimo de artefatos nesta sessão** — refatorar backlog **só Sprint 3-4 + restante Phase 1**; Phase 2/3 ficam em alto nível (respeita phase-boundary, evita retrabalho). [sequencing-critic]
2. **Fonte de verdade única** — `docs/09-backlog.md` é canônico para status; `tasks/STATUS.md` é **gerado** (`make status`), nunca editado à mão; HANDOFF = narrativa. [pm-critic #1/#15]
3. **Anti-drift leve** — `make validate-backlog` (valida `depends_on`/`covers_requirements`/`status`) rodável local + no `pre-review`. [pm-critic #3]
4. **TDD com prova real de RED** — ordem de commit (teste-RED isolado antes do código) validada no `pre-review`. [tdd-critic #1]
5. **E2E decomposto e determinístico** — fluxo crítico fatiado; Keycloak mockado **com teste compensatório** de auth real; isolamento por campus-por-teste; tiers smoke/full. [tdd-critic #2/#3/#4/#9]
6. **Paralelismo feature×processo acontece na sessão-piloto** (próxima), não nesta. Aqui só preparo o terreno. [sequencing-critic]

---

## Fase A — Gestão & Paralelismo (escopo Sprint 3-4) — #1, #2

- [ ] A1. `docs/09-backlog.md`: adicionar campos `status` `depends_on` `covers_requirements` `parallel_with` `size(S/M/L)` `offline` **apenas** às histórias de Sprint 3-4 + restante Phase 1 (E05/E06). Critérios de aceite em **Given/When/Then**. Phase 2/3 ganham só nota "detalhar no kickoff da fase". [pm #4, sequencing #4]
- [ ] A2. `docs/08-roadmap.md`: coluna **Status** nas tabelas + seção **"Modelo de Paralelização"** com caminho crítico real do que falta na Phase 1 (chain serial: Dexie v2 → useOnlineSync → pull-merge → conflito-UI; trilhas paralelas: E2E infra, reports, PWA). [sequencing #7]
- [ ] A3. `tasks/STATUS.md` **gerado** por `scripts/generate-status.sh` (lê backlog), board enxuto História→Status→Size→Depende→Paralelo. Adicionar ao `.gitignore`. [pm #15]
- [ ] A4. `Makefile` raiz: `make status` (gera STATUS.md) e `make validate-backlog` (valida IDs de `depends_on`/`covers_requirements` existem; `status` no enum). [pm #1/#3]
- [ ] A5. `.project-ai/rules/backlog-traceability.md`: declarar backlog como fonte única; exigir `covers_requirements`/`depends_on`; auditoria requisito→história no `pre-release`. [pm #1/#5]
- [ ] A6. `.project-ai/checklists/definition-of-done.md` (NOVO): DoD por história/feature/sprint (critérios verificados rodando o app, não só "código existe"). Ligar no `pre-review`. [pm #2]
- [ ] A7. `.project-ai/rules/ready-definition.md` (NOVO) + `pre-implement`: estado `ready` (critérios GWT escritos, `depends_on` todos `done`, deps de API/tabela existem) antes de `in_progress`. [pm #11]
- [ ] A8. `feature-delivery.md` + `OPERATING_MODEL.md`: documentar **orquestração paralela por subagentes** guiada pelo grafo `depends_on` (backend de uma história + frontend de outra desbloqueada). Sem worktrees. Nota: paralelismo backend×frontend está ~esgotado na Phase 1 (resto é frontend-heavy); a alavanca real agora é processo×feature e hooks independentes. [sequencing #2, decisão do usuário]

## Fase B — TDD obrigatório com prova de RED — #3

- [ ] B1. `.project-ai/rules/tdd-enforcement.md` (NOVO): ciclo Red-Green-Refactor; **prova de RED via ordem de commit** — 1º commit `test: <id> (RED)` só com arquivos de teste falhando; depois `feat: <id> (GREEN)`; depois `refactor:`. [tdd #1]
- [ ] B2. `feature-delivery.md` Fase 3: reescrever ordem como **TDD-first por camada** (teste→RED→código→GREEN→refactor), backend e frontend.
- [ ] B3. `hooks/pre-review.md`: validar via `git log` que o 1º commit que toca `*_test.go`/`*.test.ts(x)` precede o 1º commit de produção; senão bloqueia. [tdd #1] (substitui o registro em HANDOFF — cortado [pm #17])
- [ ] B4. `checklists/backend-feature-complete.md` + `frontend-feature-complete.md`: item bloqueante "teste escrito primeiro, visto falhar (ordem de commit RED→GREEN)".
- [ ] B5. `.project-ai/checklists/test-distribution.md` (NOVO): pirâmide alvo 60% unit / 30% integração / 10% E2E, para evitar ice-cream-cone. [tdd #10]

## Fase C — E2E real (tela→backend→DB), decomposto e determinístico — #4

- [ ] C1. **Infra E2E** (`frontend/e2e/`): Playwright → frontend buildado → **API Go real + Postgres real** (docker-compose de teste) → **Keycloak mockado** (token injetado). `frontend/e2e/fixtures.ts` com **isolamento por campus-por-teste** + cleanup `afterEach` via API. [tdd #4]
- [ ] C2. **Fluxo crítico decomposto** (não um E2E monolítico):
  - E2E (Playwright, fatia fina): login → criar pessoa (online) → toggle offline → pessoa visível na lista (cache) → toggle online → fila drena. Assere tela + API + Postgres.
  - **Fora do E2E** (ficam em integração/unit): pull, conflito, batching, relatórios. [tdd #3]
- [ ] C3. **Teste compensatório de auth real** `backend/internal/integration/auth_middleware_test.go` (NOVO): token válido, expirado, `email_verified` ausente→403, assinatura inválida→401 — cobre o que o mock de Keycloak no E2E não cobre. + asserção de header `Bearer` no front. [tdd #2]
- [ ] C4. `.project-ai/rules/e2e-test-tiers.md` (NOVO) + alvos: `test:e2e:smoke` (rápido, fluxo feliz) e `test:e2e:full` (gate de sprint). [tdd #9]
- [ ] C5. `.project-ai/checklists/e2e-critical-flows.md` (NOVO, gate de sprint): cada fluxo crítico com E2E full verde + auth-integration verde, antes do release. Ligar em `sprint-release.md` e `pre-release`. MSW/testcontainers seguem obrigatórios **por feature**. [tdd #2/#9]
- [ ] C6. `Makefile`/`package.json`: alvos E2E + doc de execução local. CI segue pausado.

## Fase D — Piso de cobertura frontend (credibilidade do TDD) — habilitador

- [ ] D1. Definir piso **mensurável**: `vite.config` thresholds 50% global / 80% em hooks e forms tocados pela próxima feature. Criar `frontend/src/hooks/__tests__/` com testes-semente dos hooks que o sync drainer usará (`useOfflineStatus`, store de auth, e os novos quando criados). Backfill **só o caminho da fatia**, não o codebase todo. [tdd #5]
- [ ] D2. `frontend/src/offline/__tests__/`: testes unit de fila de sync (enqueue/dequeue/batch/retry) e schema Dexie — base para o TDD do drainer. [tdd #3]

## Fase F — Pipeline de entrega autônoma (gate de revisão crítica + stop antes do push)

> Objetivo: qualquer feature futura é levada de "código" até "pronta-para-PR" **sem intervenção humana**, parando no único limite permitido (antes do `git push`).

- [ ] F1. `Makefile` raiz: alvo único **`make deliver`** que encadeia, em ordem, falhando rápido:
  1. `make validate-backlog` (gestão) → 2. gate TDD (ordem de commit RED→GREEN, ver B3) → 3. backend: `build`+`lint`+`test`+`test-integration`+`auth_middleware_test` → 4. frontend: `typecheck`+`lint`+`test`+`test:integration`+`test:coverage`(≥pisos)+`build` → 5. `test:e2e:smoke` (stack real) → 6. **gate de revisão crítica** (F2) → 7. checklist DoD (A6). Se tudo verde: imprime "READY-FOR-PR" e **PARA** (não faz push). [decisão: entrega autônoma]
- [ ] F2. **Gate de revisão crítica autônomo e bloqueante** — `.project-ai/skills/autonomous-critical-review.md` (NOVO): roda o agente `reviewer.md` (já existe) sobre o diff da branch, com saída **verificável em arquivo** `tasks/review-<branch>.md` no formato de veredito do reviewer (APPROVE/REQUEST_CHANGES/NEEDS_DISCUSSION + quality gate + clean code + complexidade + segurança). `make deliver` **falha** se o veredito não for APPROVE. Diferente do estado atual (assistivo) — agora é executável e gate de `deliver`. [decisão: revisão crítica obrigatória]
- [ ] F3. **Loop de auto-remediação (autônomo, até 3 ciclos — decisão do usuário)** — `.project-ai/workflows/autonomous-delivery.md` (NOVO): se F2 = REQUEST_CHANGES, o orquestrador aciona `refactor-for-quality` (playbook já existe), reaplica TDD (teste→RED→fix→GREEN) e re-roda `make deliver` **automaticamente, sem aval humano por ciclo**. Máx. 3 ciclos; se não convergir, registra o impasse em `tasks/review-<branch>.md` e para para decisão humana (não força merge). [autonomia com salvaguarda]
- [ ] F4. **Stop-point explícito e auditável** — `autonomous-delivery.md` declara: o pipeline NUNCA executa `git push`, `gh pr create`, `gh pr merge` (PAT sem permissão). Entrega = commits locais na branch + `READY-FOR-PR` + `tasks/review-<branch>.md` APPROVE. Última etapa imprime o comando de push **sugerido** para o humano rodar (`! git push -u origin <branch>`). [limite do usuário]
- [ ] F5. `feature-delivery.md` + `OPERATING_MODEL.md`: substituir o fim do fluxo (Fase 4.5/6) pela referência a `make deliver` como gate único autônomo; o veredito do reviewer passa a ser **produzido** (arquivo) e não só "consultado". `pre-merge.md`: apontar que o APPROVE agora vem de `tasks/review-<branch>.md`.
- [ ] F6. `CLAUDE.md`: nova seção "Autonomous Delivery & Push Boundary" — documenta `make deliver`, o gate de revisão crítica, e a proibição dura de push/PR/merge pelo agente.

## Fase E — Validação

- [ ] E1. `make build`/`make test`/`make test-integration` (back) + novo `auth_middleware_test.go` verdes.
- [ ] E2. `npm run typecheck`/`lint`/`test`/`test:integration`/`build` (front) verdes; `test:coverage` ≥ pisos.
- [ ] E3. `test:e2e:smoke` verde contra stack real (fluxo de referência da fatia C2).
- [ ] E4. `make validate-backlog` + `make status` verdes; auditoria requisito→história Phase 1 sem buracos.
- [ ] E5. **`make deliver` executa de ponta a ponta** num dry-run (sem feature nova: valida que o pipeline roda, gera `tasks/review-<branch>.md`, imprime READY-FOR-PR e NÃO faz push).
- [ ] E6. Atualizar `HANDOFF.md` + memória do projeto. Registrar a **sessão-piloto** seguinte (sync drainer sob `make deliver`, processo×feature em paralelo).

---

## Execução PARALELA deste plano (Fases A–F desta sessão)

Grafo de dependências entre as próprias tarefas do plano e mapa de fan-out por subagentes:

```
ONDA 1 (paralela — artefatos independentes, sem conflito de arquivo):
  ├─ G-PM      → A1,A2,A3,A4,A5,A6,A7  (backlog, roadmap, STATUS, Makefile-gestão, rules de PM, DoD, ready)
  ├─ G-TDD     → B1,B2,B4,B5           (regra TDD, ordem IMPLEMENT, checklists, pirâmide)
  ├─ G-E2E-be  → C3                    (auth_middleware_test.go — backend, isolado)
  └─ G-COV-fe  → D1,D2                 (pisos vite + testes-semente offline/hooks — frontend, isolado)

ONDA 2 (depende da onda 1 — infra E2E + autonomia consomem alvos/saídas da onda 1):
  ├─ G-E2E-infra → C1,C2,C4,C5,C6      (Playwright, fixtures, tiers, checklist E2E, package.json)
  └─ G-AUTONOMY  → B3,F1,F2,F3,F4,F5,F6 (gate TDD por commit, make deliver, gate de revisão, stop-point, docs)
        ▲ depende de: alvos de teste (G-PM/G-TDD/G-E2E), pois `make deliver` os encadeia

ONDA 3 (serial — consolidação; eu mesmo, não subagente):
  └─ E1..E6  validação integrada + dry-run de `make deliver` + HANDOFF/memória
```

Regras de paralelismo (evitar colisão de escrita):
- Cada subagente escreve em **conjunto de arquivos disjunto** (sem dois agentes no mesmo arquivo). `Makefile` raiz: G-PM cria o esqueleto + alvos de gestão; G-AUTONOMY (onda 2) **edita** para somar `deliver` — serial por isso.
- `package.json`/`vite.config`: só G-COV-fe (onda 1) e G-E2E-infra (onda 2) tocam, em ondas diferentes → sem corrida.
- `feature-delivery.md`/`OPERATING_MODEL.md`: tocados por G-TDD (onda 1, seção IMPLEMENT) e G-AUTONOMY (onda 2, seção fim-de-fluxo) → ondas diferentes, seções diferentes.
- Sem git worktrees (todos no mesmo working tree, coordenados por onda).
- Eu consolido cada onda antes de disparar a próxima, e rodo a Fase E sozinho.

---

## Incorporado dos críticos (rastreabilidade)
- **PM**: SoT única+STATUS gerado (#1/#15), DoD (#2), anti-drift `make validate-backlog` (#3), sizing S/M/L (#4), auditoria de requisitos (#5), ready-definition (#11). **Adiados p/ piloto/Phase 2**: retrospectiva (#7), ADR cadence (#8), bug-triage (#9), UAT gate (#10), scope-change (#12).
- **TDD/E2E**: prova de RED por commit (#1), boundary Keycloak + auth test (#2), E2E decomposto (#3), seeding/isolamento (#4), piso de cobertura (#5), tiers smoke/full (#9), pirâmide (#10). **Adiados**: flaky-policy (#6), contract tests (#7), mutation testing (#8) → Phase 2.
- **Sequencing**: escopo backlog só Sprint 3-4 (#1/#4); paralelismo real = processo×feature, na sessão-piloto (#2/#5); caminho crítico no roadmap (#7).

## Fora de escopo nesta sessão
- Implementar o sync drainer (vem na **sessão-piloto** seguinte, sob as novas regras, com processo×feature em paralelo).
- Reativar CI (pausado). Detalhar Phase 2/3. Retrospectiva/ADR/bug-triage/UAT/flaky/mutation/contract (Phase 2).

## Review (sessão concluída — 2026-06-28/29)

### Execução
Plano v3 executado em 3 ondas paralelas + validação serial. Antes de executar, o plano foi submetido a **3 críticos adversariais** (PM/processo, TDD/E2E, sequenciamento) e revisado (v1→v3) incorporando: escopo de backlog só Sprint 3-4 (evita retrabalho Phase 2/3), fonte de verdade única (backlog; STATUS gerado), prova de RED por ordem de commit, boundary de Keycloak com teste compensatório, E2E decomposto e determinístico, e pisos de cobertura mensuráveis.

- **Onda 1** (4 subagentes paralelos): G-PM (backlog/roadmap/STATUS/Makefile-gestão/rules/DoD/ready), G-TDD (regra TDD/IMPLEMENT/checklists/pirâmide), G-E2E-be (`auth_middleware_test.go`), G-COV-fe (pisos + 32 testes-semente).
- **Onda 2** (2 subagentes, serial entre si): G-E2E-infra (Playwright + mock-OIDC real + docker-compose.e2e + tiers + smoke honesto), G-AUTONOMY (`make deliver` + gate de revisão crítica + auto-remediação + boundary).
- **Onda 3** (eu, serial): validação integrada — corrigi 2 bloqueadores **pré-existentes** (eslint `react-hooks/use-memo` em ReportsPage; `baseUrl` deprecated no tsconfig sob TS 6.0.3) e o include/exclude do vitest (e2e fora do unit runner); `go mod tidy` (go-jose direto); endureci o `deliver-review-gate` (último heading canônico, fail-closed comprovado).

### Resultado da revisão crítica autônoma
Agente revisor executado sobre todo o diff → **APPROVE** (0 BLOCKER, 0 MAJOR, 5 MINOR, 1 SUGGESTION). Confirmou independentemente: auth test é real (não falso-GREEN), push boundary hermética, gate fail-closed. 1 MINOR de segurança do próprio gate foi **remediado** na hora.

### Validação (tudo verde, sem Docker)
`make validate-backlog` OK · `make -n deliver` parse limpo (7 gates) · `make deliver-review-gate` fail-closed comprovado (falha sem APPROVE, passa com APPROVE, rejeita citação solta) · backend build + `go test -short` (9 pkgs) + auth test `-race` · frontend tsc + 44 unit + 8 integração + **0 erros eslint** + build PWA.

### Limite respeitado
Nada commitado/pushado. Tudo no working tree. O agente NUNCA roda `git push`/`gh pr create`/`gh pr merge` — `make deliver` só **sugere** o push para o humano.

### Pendências (para a sessão-piloto)
- Steps Docker-gated (integração backend, e2e:smoke) precisam de Docker ligado no gate de sprint.
- E2E offline está `test.fixme` até o drainer ser ligado na app.
- Operador: `cd frontend && npm install` + `npx playwright install chromium` antes do E2E.
- Piso global de cobertura (50%) sobe incrementalmente; testes runtime de Dexie v1→v2 (S05.1) na piloto.

### Status das tarefas
A1–A7 ✅ · B1–B5 ✅ · C1–C6 ✅ · D1–D2 ✅ · F1–F6 ✅ · E1–E6 ✅ (E5 = `make -n deliver` + gate dry-run; suítes não-Docker verdes). Próximo: **sessão-piloto** do sync drainer sob `make deliver`.
