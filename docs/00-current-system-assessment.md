# 00 - Current System Assessment

## 1. What This Repository Is

**SOS-Gestao-Final** is a Django 5.2.6 web application designed for social service management at an NGO ("ONG"). It provides an admin-centric interface for recording service attendance, managing users and permissions, tracking employee hours, and managing expenses.

The system is built around **Django Admin** with **Jazzmin** theming and does not include a custom frontend, API layer, or mobile interface.

---

## 2. Current Business Purpose

The application supports basic operational management of a social assistance organization:

- **Beneficiary registration** (Pessoa model): name, CPF, birth date, gender, neighborhood, church affiliation
- **Service attendance** (Atendimento model): links a person to services, a professional, complaints, and observations
- **Service type catalog** (TipoServico model): categorizes available services
- **Expense tracking** (DespesaONG model): records expenses by category with audit trail
- **Time tracking** (RegistroPonto model): employee punch records with GPS location
- **User and permission management**: 5-tier role system with 17 custom permissions and full audit logging

---

## 3. Architecture Summary

### Stack
| Layer | Technology |
|-------|-----------|
| Language | Python 3.12 |
| Framework | Django 5.2.6 |
| UI | Django Admin + Jazzmin 3.0.1 |
| Database (dev) | SQLite3 |
| Database (prod) | PostgreSQL 15 (via Docker) |
| Server | Gunicorn 23.0.0 |
| Container | Docker + Docker Compose |
| CI/CD | GitHub Actions |

### Django Apps
| App | Purpose |
|-----|---------|
| `ong_manager` | Project root, settings, dashboard, time tracking, expenses |
| `atendimento` | Beneficiary registration, service types, attendance records |
| `users` | User profiles, custom permissions, access grants, audit logging |

### Key Dependencies
```
Django==5.2.6
django-jazzmin==3.0.1
django-braces==1.17.0
psycopg2-binary==2.9.11
gunicorn==23.0.0
pandas==2.3.3
numpy==2.2.6
plotly==6.3.1
scipy==1.15.3
```

### Architecture Pattern
- **Admin-centric**: All data management happens through Django Admin with customized views
- **Monolithic**: Single Django project with three apps, no service separation
- **Server-rendered**: No frontend framework; Jazzmin provides the UI
- **No API layer**: No REST or GraphQL endpoints exist

---

## 4. Dependency Analysis

### Appropriate Dependencies
- **Django 5.2.6**: Mature, well-documented framework
- **psycopg2-binary**: Standard PostgreSQL adapter
- **gunicorn**: Production-ready WSGI server
- **django-braces**: Lightweight view mixins

### Questionable Dependencies
- **pandas 2.3.3 + numpy 2.2.6 + scipy 1.15.3**: Heavy data science libraries used only for a single basic report view. These add ~200MB to the container image and significant attack surface for minimal value.
- **plotly 6.3.1**: Full plotting library when the report template only has a placeholder chart area.
- **django-jazzmin 3.0.1**: Only useful if Django Admin remains the primary UI. Becomes dead weight with a custom frontend.

### Missing Dependencies
- No REST framework (django-rest-framework or similar)
- No CORS handling
- No JWT/token authentication
- No file storage backend (S3, etc.)
- No task queue (Celery, etc.)
- No caching layer (Redis, etc.)

---

## 5. Codebase Strengths

1. **Well-designed permission system**: Three-tier model with PerfilUsuario, PermissaoCustomizada, AcessoUsuario, and AuditoriaAcesso. Covers 5 modules with 17 granular permissions. Reusable mixins and decorators.

2. **Audit logging foundation**: AuditoriaAcesso model tracks user actions, IP addresses, user agents, success/failure. This is a strong foundation for LGPD compliance.

3. **Management commands**: `setup_permissoes` and `criar_profissional` automate initial setup. Good pattern for repeatable deployments.

4. **Dockerized infrastructure**: Docker Compose with PostgreSQL is production-ready with minor hardening.

5. **CI/CD pipeline**: GitHub Actions runs migrations and tests on every push/PR.

6. **Signal-based auto-creation**: User profiles are auto-created via Django signals, reducing manual setup.

7. **Comprehensive existing documentation**: 3,500+ lines of documentation in Portuguese covering security, implementation, and operations.

---

## 6. Codebase Weaknesses

### Critical Gaps

| Gap | Impact |
|-----|--------|
| **No API layer** | Cannot support mobile apps, PWA, or third-party integrations |
| **No custom frontend** | Admin-only UI is unusable for volunteers with low technical skill |
| **No offline support** | Cannot operate during field events without connectivity |
| **No file/document storage** | Cannot attach documents, consent forms, or exams |
| **No campaign/event model** | Cannot track social actions, events, or campaigns |
| **No donation model** | Cannot manage financial or in-kind donations |
| **No triage model** | Cannot record initial screening or service workflow |
| **No workflow engine** | No state machine for attendance lifecycle |
| **No consent/LGPD model** | No digital consent capture or data deletion support |
| **No multi-tenancy/campus** | Cannot segregate data by church campus or region |

### Security Issues
- `SECRET_KEY` hardcoded in settings.py
- `DEBUG = True` in committed code
- `ALLOWED_HOSTS = ['*']`
- `.env` file present with credentials (should be gitignored)
- No HTTPS enforcement
- No rate limiting
- No CORS configuration
- No Content Security Policy

### Data Model Limitations
- `Pessoa` model is minimal (no address fields beyond neighborhood, no phone, no email)
- No person role system (same person cannot be both volunteer and beneficiary)
- No family/household grouping
- No document attachment support
- Attendance model lacks workflow states
- No referral or follow-up tracking

### Testing
- `atendimento/tests.py` is empty
- `ong_manager` has no tests
- Only `users/tests.py` has 9 test cases
- No integration tests, no API tests, no end-to-end tests

### Code Quality
- Views are minimal (dashboard + single report)
- Report view uses pandas/numpy/scipy for simple aggregations that could use Django ORM
- Template structure is inconsistent (nested `relatorio.html/Relatorio.html`)
- No form classes defined
- No serializers
- No API documentation

---

## 7. Reuse vs. Rebuild Recommendation

### Verdict: **Rebuild with selective knowledge transfer**

The current codebase cannot serve as the foundation for the v2 platform. The gaps are not incremental — they are structural:

1. **The UI paradigm must change.** Django Admin is not suitable for field volunteers on mobile devices. A React PWA requires a complete API layer that does not exist.

2. **The data model must be redesigned.** The unified person registry with multi-role support, workflow states, consent management, campaign association, and offline sync capability require a fundamentally different schema.

3. **The technology stack should change.** The user's preferred stack (Go backend + React frontend) is better suited for the requirements: lower resource usage, faster API responses, simpler deployment, native concurrency for sync operations, and strong typing for a complex domain model.

4. **Offline-first is architectural, not additive.** You cannot bolt offline sync onto a Django Admin application. It requires API-first design, conflict resolution, local storage strategy, and sync protocols from day one.

### What to Carry Forward

| Asset | How to Reuse |
|-------|-------------|
| Permission model design | Port the 3-tier permission concept (profile → permission → audit) to the Go backend |
| Audit logging pattern | Replicate the AuditoriaAcesso approach in the new system |
| Permission definitions | Reuse the 17-permission taxonomy across 5 modules as a starting point for RBAC |
| Management commands pattern | Create equivalent CLI tools in Go for setup/seeding |
| Business domain knowledge | Use existing models as reference for the v2 domain model |
| Docker/CI patterns | Adapt the Docker and GitHub Actions setup for the Go+React stack |

### What to Discard

- Django Admin UI and Jazzmin customization (replaced by React frontend)
- pandas/numpy/scipy/plotly dependencies (replaced by Go-native reporting or lightweight JS charts)
- Django views and templates (replaced by API + React)
- Django ORM models (replaced by Go data layer)
- SQLite development database (Go apps typically use PostgreSQL directly)

### Migration Path

1. **Phase 0**: Complete this documentation suite (current task)
2. **Phase 1**: Build Go API with core domain models, auth, and RBAC
3. **Phase 2**: Build React PWA with offline-first architecture
4. **Phase 3**: Migrate existing data from SQLite/PostgreSQL to new schema
5. **Phase 4**: Decommission Django application

The Django application can remain operational during migration. Data migration scripts should be written to transfer Pessoa, Atendimento, and User records to the new system.

---

## 8. Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Existing data loss during migration | High | Write migration scripts with validation; keep Django running in parallel |
| Volunteer resistance to new UI | Medium | Involve volunteers in UI testing early; keep flows simple |
| Scope creep during rebuild | High | Strict MVP definition; phase features incrementally |
| Offline sync complexity | High | Use proven patterns (CRDTs or last-write-wins); start with simple sync |
| Single developer bottleneck | Medium | AI-assisted development with clear specs reduces dependency |
| Budget constraints | High | Use free-tier cloud services; open-source stack; no licensing costs |
