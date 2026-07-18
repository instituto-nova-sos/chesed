# 21 — Proposta: Gateway de Pagamento e Doações Online (v2)

> **Status:** Proposta (não implementado). Fase 2.
> **Autor:** Pedro Barbosa · **Perspectiva:** engenharia sênior (Go)
> **Pré-requisito de leitura:** [10-data-model.md](10-data-model.md) (tabela `donation`), [13-security-and-compliance.md](13-security-and-compliance.md), [05-architecture-proposal.md](05-architecture-proposal.md).

---

## 1. Contexto e problema

Na **v1**, o módulo de Doações é um **registro manual** (livro-caixa): a secretaria lança
doações **já recebidas** (dinheiro, bens, serviços) e o sistema emite recibo em PDF.
**Não há cobrança online** — nenhum gateway, PIX, cartão ou boleto.

A **v2** propõe **arrecadação online**: o doador paga pelo próprio sistema (ou por um link
público), e a doação é **conciliada automaticamente** — sem digitação manual, com recibo
emitido no ato da confirmação.

### Princípio norteador (precdeito Felipe: sem duplicação / sem erro de mesclagem)
- **Reusar, não duplicar:** a tabela `donation` e o gerador de recibo PDF **permanecem**.
  A cobrança online é uma **camada nova** que, ao confirmar, **preenche uma `donation`** —
  o mesmo destino do fluxo manual.
- **Isolamento de módulo:** todo o código de pagamento vive em `internal/payment/`
  (novo pacote) + migrations **novas** (não altera as existentes) → zero conflito de merge
  com o que já está em produção.

---

## 2. Objetivo e escopo

| Em escopo (v2) | Fora de escopo |
|---|---|
| Doação avulsa online via **PIX** (prioritário BR), **cartão** e **boleto** | Doação recorrente/assinatura (v2.1) |
| **Link/página pública** de doação (com ou sem login) | Split de pagamento entre campi |
| **Webhook** de confirmação + conciliação automática | Antifraude próprio (delegado ao PSP) |
| Recibo automático (reusa PDF da v1) | Emissão fiscal/NF-e |
| Trocar de provedor sem reescrever o app | Carteira/saldo interno |

---

## 3. Arquitetura (Ports & Adapters — o ponto central)

O erro clássico é acoplar o código ao SDK de **um** gateway. Como as taxas e a
disponibilidade de PSP mudam (e uma ONG troca de provedor por custo), o design correto é
uma **porta** (interface) no domínio e **adaptadores** por provedor.

```
handler/payment  →  service/payment  →  PaymentGateway (interface, porta)
                                         ├── adapter: EfiAdapter      (PIX/boleto)
                                         ├── adapter: MercadoPago     (cartão/PIX)
                                         └── adapter: StripeAdapter   (cartão intl)
                          │
                          └→ repository: PaymentTransactionRepository → Postgres
```

Interface mínima (o app conhece **só isto** — segue o padrão de interfaces já usado em
`service/`):

```go
// internal/payment/gateway.go
package payment

// PaymentGateway is the port. Each PSP is an adapter. The app never imports a
// provider SDK outside its adapter file — swapping providers touches one file.
type PaymentGateway interface {
    // CreateCharge starts a charge and returns provider ids + the payer-facing
    // artifact (PIX copia-e-cola / QR, boleto line, or hosted-checkout URL).
    CreateCharge(ctx context.Context, in ChargeInput) (*Charge, error)

    // ParseWebhook validates the provider signature and normalizes the event.
    // Signature verification is mandatory — never trust an unsigned callback.
    ParseWebhook(ctx context.Context, headers http.Header, body []byte) (*WebhookEvent, error)

    Provider() string // "efi" | "mercadopago" | "stripe"
}
```

**Por que sênior:** a regra de negócio (criar doação, emitir recibo, auditar) fica
**independente do provedor**. Testes usam um `FakeGateway`. Sem `if provider == ...`
espalhado — a escolha é injeção de dependência (composition root em `cmd/server`).

---

## 4. Modelo de dados (estender, não duplicar)

`donation` **não muda de forma disruptiva**. Adiciona-se **uma** tabela e **uma** coluna:

```sql
-- migration nova (não altera as existentes)
CREATE TABLE payment_transaction (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    donation_id       UUID REFERENCES donation(id),        -- nulo até confirmar
    campus_id         UUID NOT NULL REFERENCES campus(id), -- RLS scoping (padrão existente)
    provider          VARCHAR(20)  NOT NULL,               -- efi | mercadopago | stripe
    provider_charge_id VARCHAR(120) NOT NULL,
    method            VARCHAR(10)  NOT NULL,               -- PIX | CARD | BOLETO
    amount            NUMERIC(12,2) NOT NULL,
    currency          VARCHAR(3)   NOT NULL DEFAULT 'BRL',
    status            VARCHAR(20)  NOT NULL,               -- PENDING|PAID|FAILED|EXPIRED|REFUNDED
    idempotency_key   UUID NOT NULL,                       -- 1 intenção = 1 cobrança
    payer_name        VARCHAR(200),
    payer_email       VARCHAR(200),
    paid_at           TIMESTAMPTZ,
    raw_last_event    JSONB,                               -- auditoria do provedor
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_provider_charge UNIQUE (provider, provider_charge_id),
    CONSTRAINT uq_payment_idempotency UNIQUE (idempotency_key)
);
-- donation ganha só o vínculo reverso (opcional) — origem da doação:
ALTER TABLE donation ADD COLUMN payment_transaction_id UUID REFERENCES payment_transaction(id);
```

**Estado como máquina** (reusa o padrão de transições já existente em `attendance`):
`PENDING → PAID | FAILED | EXPIRED`; `PAID → REFUNDED`. Só `PAID` cria/preenche a `donation`.

---

## 5. Fluxo (com idempotência e conciliação)

```
1. Doador abre a página pública de doação → escolhe valor + método
2. POST /api/v1/donations/checkout  (idempotency_key gerado no client)
      → service cria payment_transaction (PENDING) → gateway.CreateCharge
      → devolve PIX copia-e-cola/QR (ou URL de checkout hospedado)
3. Doador paga no banco/app
4. PSP → POST /api/v1/webhooks/payments/{provider}   (assinado)
      → ParseWebhook valida assinatura → localiza transação por provider_charge_id
      → status PAID → cria donation (mesmo shape da v1) → emite recibo PDF → audita
5. Página faz polling em GET /donations/checkout/{id}/status  (ou SSE) → "confirmado"
```

**Pontos que separam júnior de sênior aqui:**
- **Idempotência dupla:** no checkout (`idempotency_key`) e no webhook (o PSP reenvia o
  mesmo evento; `provider_charge_id` único evita doação duplicada). Isso resolve o mesmo
  risco de request duplicado tratado no offline-sync.
- **Webhook é a fonte da verdade**, não o retorno do browser (o usuário pode fechar a aba).
- **Nunca** confiar em valor vindo do client no webhook — reler o valor do PSP.
- **Verificação de assinatura** obrigatória (HMAC/JWT do provedor) antes de processar.

---

## 6. Opções de provedor — pagas e gratuitas

> Taxas são de referência (mudam); **validar no momento da contratação**. ONG costuma
> conseguir **tarifa social** — vale negociar.

### 6.1 Gratuitas / custo mínimo
| Opção | Como | Prós | Contras |
|---|---|---|---|
| **PIX estático** (QR fixo / copia-e-cola do próprio banco) | Sem gateway; gera o BR Code no backend | **Custo zero**, sem intermediário | **Sem conciliação automática** (não sabe quem pagou) → volta a lançar manual; sem cartão/boleto |
| **PIX dinâmico via PSP com plano grátis** (ex.: Efí/Asaas em faixas isentas) | API do PSP | Conciliação automática via webhook | Precisa de conta PJ + certificado; isenção só até certo volume |
| **Open-source (self-host)** | Nenhum gateway BR maduro 100% OSS que liquide dinheiro; PIX ainda exige um PSP autorizado (BACEN) | Controle | Você ainda precisa de um PSP para liquidar — "grátis" só na camada de software |

### 6.2 Pagas (gateway completo, recomendado para conciliação)
| Provedor | PIX | Cartão | Boleto | Observações p/ ONG |
|---|---|---|---|---|
| **Efí (ex-Gerencianet)** | ✅ excelente | ✅ | ✅ | PIX barato, API boa, popular em ONG/igreja; certificado mTLS |
| **Asaas** | ✅ | ✅ | ✅ | Foco em PJ pequena/ONG, split, régua de cobrança |
| **Mercado Pago** | ✅ | ✅ forte | ✅ | Checkout pronto, grande alcance, sandbox bom |
| **PagBank (PagSeguro)** | ✅ | ✅ | ✅ | Ampla aceitação de cartão |
| **Stripe** | parcial (via parceiro) | ✅ internacional | ❌ | Ótimo p/ doador no exterior; PIX limitado no BR |
| **Pagar.me** | ✅ | ✅ | ✅ | Boa API dev, subsidiária Stone |

### 6.3 Recomendação de engenharia
- **MVP v2:** **PIX dinâmico** por um PSP ONG-friendly (**Efí** ou **Asaas**) como primeiro
  adapter — é onde está o volume no Brasil e a taxa é a menor.
- **+ Cartão** via **Mercado Pago** (checkout hospedado → **fora do escopo PCI**).
- A interface `PaymentGateway` permite começar com **um** e adicionar os outros sem
  reescrever o app. Não escolher "o gateway definitivo" agora — escolher a **abstração**.

---

## 7. Segurança e conformidade (não negociável)

- **PCI-DSS:** **não** trafegar/armazenar dados de cartão no nosso backend. Usar
  **checkout hospedado / tokenização** do PSP → o app nunca vê o PAN. (Alinha com
  [13-security-and-compliance.md](13-security-and-compliance.md).)
- **Segredos do PSP** (client_id/secret, certificado mTLS) **só em env/secret manager** —
  nunca no código nem no `realm-export`/imagem (regra CLAUDE.md).
- **Webhook:** validar assinatura; endpoint idempotente; rate-limit (já há `httprate`).
- **LGPD:** `payer_email`/`payer_name` são PII → mesmas regras de retenção/anonimização;
  não logar em claro.
- **Auditoria:** toda transição de `payment_transaction` grava em `audit_log` (reusa
  `AuditService` existente).
- **Campus scoping:** `payment_transaction.campus_id` sob a mesma RLS (`CampusTx`) já usada.

---

## 8. Fases de entrega (incremental, sem big-bang)

1. **v2.0 — PIX dinâmico (1 adapter):** tabela + interface + EfiAdapter + webhook +
   página pública + recibo automático. Entrega o essencial.
2. **v2.1 — Cartão + boleto:** MercadoPago adapter (checkout hospedado).
3. **v2.2 — Recorrência:** doação mensal (assinatura do PSP).
4. **v2.3 — Painel:** conciliação, estornos, relatório de arrecadação (reusa
   `reports`/`dashboards` da v1).

---

## 9. Checklist anti-duplicação / anti-merge (preceito Felipe)

- [ ] Nenhuma alteração destrutiva em `donation` (só coluna nova opcional).
- [ ] Código novo isolado em `internal/payment/` + `frontend/src/pages/DonatePublic`.
- [ ] Migrations **novas** e sequenciais (000034+), com `.up`/`.down`.
- [ ] Reusar `AuditService`, gerador de recibo PDF e RLS/`CampusTx` — **não** recriar.
- [ ] Interface + `FakeGateway` para testes (unit) + integração do webhook (testcontainers).
- [ ] Um provedor por PR (Efí primeiro) → PRs pequenos, revisão limpa, sem conflito.

---

## 10. Decisões a levar para a call

1. **Vamos cobrar online?** (muda LGPD, PCI e contratação de PSP)
2. **Qual PSP primário?** Recomendação técnica: **Efí ou Asaas** (PIX barato, ONG-friendly).
3. **Página pública de doação** exige domínio + deploy definido (pauta do deploy/Bruno).
4. **Meta de custo:** PIX ~= centavos/tx; cartão ~= 3–5% + fixo — definir se repassa taxa.
