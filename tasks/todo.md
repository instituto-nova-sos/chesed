# Termo de Voluntário — Aceite por Assinatura Digital OU Documento Anexado (self-service)

## Objetivo
Na própria tela do Termo de Voluntário, o voluntário escolhe **um** de dois caminhos para
aceitar, ambos self-service:
1. **Assinatura digital** desenhada (canvas → PNG base64), igual ao Consentimento.
2. **Anexar documento** do termo assinado (imagem JPEG/PNG **ou** PDF).

Escopo coberto por **RF-07** ("registering signed consents **and terms**; Digital signature
capture on mobile"). O `volunteer_agreement` já modela `signature_method ∈ {DIGITAL,
MANUAL_UPLOAD}` — hoje só o coordenador anexa (na ficha da pessoa) e o digital é um clique
sem assinatura desenhada. Este trabalho torna os dois caminhos self-service e adiciona a
assinatura desenhada real.

## Contexto verificado no código
- `SignaturePadCanvas.tsx` (em `components/consents/`) é reutilizável (canvas hand-rolled →
  `toDataURL('image/png')`). Consent persiste em `consent.signature_data TEXT`.
- Aceite digital: `POST /volunteer-agreement/accept` (self-service, personID do token) →
  `svc.AcceptDigital(personID, ip, ua)` → `repo.AcceptDigital` grava `signature_method=
  'DIGITAL'` mas **não** guarda imagem. `volunteer_agreement` **não tem** coluna de imagem.
- Upload manual: `POST /persons/{id}/agreement/upload` — **RBAC coordenador**, `{id}` da URL
  (não serve p/ self-service seguro). Reusa `svc.UploadManual(personID, filePath)`; salva em
  `uploadDir/{campus}/{person}/{uuid}{ext}`; MIME whitelist PDF/JPEG/PNG, 10 MB.
- Última migration = `000032` → próxima = **000033**.
- Guardrails: campus scoping OK (agreement é campus-bound via person); mutação já audita.
  CI pausado → gates locais.

## Plano (TDD: RED → GREEN → REFACTOR por camada)

### Backend — persistência da assinatura desenhada
- [ ] **Migration 000033** `add_agreement_signature_data`
  - `.up.sql`: `ALTER TABLE volunteer_agreement ADD COLUMN signature_data TEXT;`
  - `.down.sql`: `ALTER TABLE volunteer_agreement DROP COLUMN signature_data;`
- [ ] **Domain** `volunteer_agreement.go`: `SignatureData *string json:"signature_data"`.
- [ ] **Repository** `volunteer_agreement_repository.go`:
  - `AcceptDigital(..., signatureData string)`: grava `signature_data`; incluir no
    `RETURNING`.
  - Adicionar `signature_data` a todos os `SELECT`/`RETURNING` e aos scanners
    (`scanAgreement`, `scanAgreementRow`).
- [ ] **Service** `AcceptDigital(ctx, personID, ip, ua, signatureData)`:
  - Validar `signatureData` não-vazio → `ErrValidation`. Audit `NewValues` **sem** a imagem.

### Backend — upload self-service (voluntário anexa o próprio termo)
- [ ] **Novo handler** `AcceptUpload` → `POST /volunteer-agreement/upload` (multipart
  `document`), **personID do token** (não da URL), mesmas validações do `Upload` existente
  (MIME PDF/JPEG/PNG, 10 MB, path seguro). Reusa `svc.UploadManual(personID, filePath)`.
  - `claims.PersonID == Nil` → 400. Salvar em `uploadDir/{campus}/{person}/...`.
- [ ] **Rota**: registrar em `registerAgreementRoutes` (grupo self-service, sem
  `RequireAgreement`), ao lado de `/accept` e `/reject`.
- [ ] **Endpoint do coordenador permanece intacto** (`/persons/{id}/agreement/upload`).

### Backend — handler accept (assinatura no body)
- [ ] `Accept`: decodificar `{ "signature_data": string }`, `validateStruct`; vazio → 400.
  Repassar ao service.

### Tests backend (RED primeiro)
- [ ] `volunteer_agreement_service_test.go`: accept com assinatura persiste; sem → validação.
- [ ] **Novo** `handler/volunteer_agreement_test.go`: accept 400 sem assinatura / 200 com;
  upload self-service 200 / 400 sem person / 400 MIME inválido.
- [ ] **Integração** `internal/integration/volunteer_agreement_test.go`
  (`//go:build integration`): assinatura gravada no DB; 400 sem assinatura; upload
  self-service grava `document_path` + `signature_method='MANUAL_UPLOAD'`; 409 já aceito;
  404 sem pending; campus boundary. (Mandato de integração do CLAUDE.md.)

### Frontend
- [ ] **Reuso**: mover `SignaturePadCanvas` → `components/ui/SignaturePadCanvas.tsx`
  (genérico); reapontar import em `ConsentFormPage.tsx` + seus testes.
- [ ] **Types** `types/person.ts`: `VolunteerAgreement.signature_data?: string | null`.
- [ ] **API client** `api/persons.ts`:
  - `acceptAgreement(signatureData: string)` → body `{ signature_data }`.
  - **Nova** `uploadAgreementSelf(file: File)` → `POST /volunteer-agreement/upload`
    (multipart, via `apiClientRaw`).
- [ ] **Página** `VolunteerAgreementPage.tsx` — seletor de método (rádio/toggle):
  - **Assinar digitalmente**: `SignaturePadCanvas`; "Aceitar Termo" desabilitado até assinar;
    `acceptAgreement(signature)`.
  - **Anexar documento**: input de arquivo (JPEG/PNG/PDF, ≤10 MB, validação client-side igual
    ao `AgreementUploadModal`); "Aceitar Termo" desabilitado até escolher arquivo;
    `uploadAgreementSelf(file)`.
  - Ambos os sucessos: forçar re-login Keycloak (como hoje) p/ refrescar claims.
- [ ] Reusar validação de arquivo do `AgreementUploadModal` (extrair helper compartilhado se
  reduzir duplicação; senão, replicar mínimo).

### Tests frontend (RED primeiro)
- [ ] **Novo** `pages/__tests__/VolunteerAgreementPage.test.tsx`:
  - método "assinar": bloqueia aceite sem assinatura; envia `signature_data`.
  - método "anexar": bloqueia sem arquivo; rejeita MIME inválido; chama `uploadAgreementSelf`.
  - alternância entre métodos limpa o estado do outro.
- [ ] Integração `__integration__/`: `acceptAgreement` serializa `signature_data`;
  `uploadAgreementSelf` monta multipart no endpoint certo.

### Docs
- [ ] `docs/11-api-design.md`: body de `/volunteer-agreement/accept` (`signature_data`
  obrigatório p/ DIGITAL) + novo `POST /volunteer-agreement/upload` self-service.
- [ ] `docs/10-data-model.md`: `signature_data` no DDL + nota migration 000033.
- [ ] `docs/16-iam-and-access-control.md`: registrar que upload passa a ter também um caminho
  self-service (personID do token) além do endpoint do coordenador.

## Fora de escopo
- Não mexer no endpoint de upload do coordenador nem no `AgreementGuard` (dead code).
- Não corrigir doc-drift pré-existente de `volunteer_agreement` (só anotar).
- Não alterar índice UNIQUE nem política RLS.

## Verificação (gates locais — CI pausado)
- [ ] Backend: `make migrate-up`, `make build`, `make lint`, `make test`,
      `make test-integration`.
- [ ] Frontend: `npm run typecheck`, `npm run lint`, `npm test`, `npm run build`.
- [ ] Smoke manual na stack local: aceitar via assinatura (checar `signature_data` no DB) e
      via anexo (checar `document_path` + arquivo salvo); audit log em ambos.

## Review section

**Delivered.** Volunteer agreement now accepts either a drawn digital signature OR a
self-service document upload (image/PDF), mirroring the consent flow. RF-07.

Backend:
- Migration `000033` adds nullable `signature_data TEXT` (up+down, roundtrip-verified).
- Domain/repo/service/handler thread the drawn signature; accept requires a non-empty
  signature (400 otherwise). New self-service `POST /volunteer-agreement/upload` takes the
  person from the token (coordinator endpoint untouched).

Pre-existing bug FIXED (surfaced by the first agreement integration test): `accepted_by_user`
/`uploaded_by` are FKs to `app_user.id`, but the code wrote the Keycloak subject → FK 23503
→ 500 in prod. Threaded `app_user.id` into `AuthClaims` via `AutoProvision`; repo binds NULL
for the zero UUID. Only `volunteer_agreement` had actor FKs to `app_user`, so scope stayed
contained. Locked by integration assertions.

Frontend:
- `SignaturePadCanvas` moved `components/consents/` → `components/ui/` (generic; consent
  repointed). New `useAcceptMethod` hook + `AgreementComponents` + shared `agreementFile`
  validator. Page has a method selector; "Aceitar Termo" disabled until signed or file chosen.

Verification (all green): backend build/lint/`go test ./...`/full integration suite (171s);
frontend typecheck/lint/364 tests/build; migration up/down roundtrip; **real-stack smoke**:
accept без assinatura→400, com assinatura→200 with `signature_data` persisted and
`accepted_by_user = app_user.id` (FK fix proven E2E); direct-grant reverted, smoke data +
Keycloak attribute cleaned up. Added `backend/uploads/` to `.gitignore` (runtime artifact).

Also in this branch (prior dev fixes): `init-realm.sh` sslRequired + campus guards, README
Docker build-flow section, "Nova Triagem" button.

NOT pushed (hard boundary). READY for a human to push/PR.
