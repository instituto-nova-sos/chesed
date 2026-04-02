# 01 - Product Vision

## Vision Statement

**SOS Gestao** is an integrated management platform for **Instituto Nova SOS** that centralizes the registration, triage, attendance, and follow-up of beneficiaries across social service campaigns and events — enabling data-driven decision-making, LGPD compliance, and offline field operations.

---

## Business Context

**Instituto Nova SOS** is a social organization linked to a church, focused on community support through educational, social, assistance, and human development activities. The organization operates across multiple campuses/regions (Brazil, USA, Europe) and depends heavily on volunteer labor.

### Current State
- Operations are managed manually or through fragmented tools
- Beneficiary data is not centralized
- Service history is difficult to consolidate
- Reporting is limited and labor-intensive
- Data privacy and security are not systematically enforced
- Field events lack digital tools for data collection

### Desired State
- Single source of truth for all people (beneficiaries, volunteers, professionals)
- Complete attendance history from triage to follow-up completion
- Campaign and event management with team coordination
- Donation tracking and accountability
- Management reports and social impact metrics
- LGPD-compliant consent and data handling
- Offline-capable mobile interface for field operations

---

## Target Users

### Primary Users

| Role | Description | Key Needs |
|------|------------|-----------|
| **Volunteer** | Community members assisting during events and campaigns | Simple mobile interface, offline data entry, quick triage forms |
| **Secretary** | Administrative staff managing day-to-day operations | Beneficiary registration, scheduling, report generation |
| **Professional** | Health, legal, social workers providing specialized services | Attendance recording, history review, document attachment |
| **Coordinator** | Team leads managing campaigns, events, and volunteer teams | Campaign setup, team assignment, progress dashboards |
| **Administrator** | System administrators managing users, permissions, and data | User management, RBAC configuration, audit review, data export |

### Secondary Users
- **Beneficiaries**: Indirect users who benefit from accurate record-keeping and follow-up
- **Donors**: Indirect users who receive donation receipts and transparency reports
- **Church leadership**: Consumers of management reports and social impact metrics

---

## Product Principles

1. **Simplicity over power**: Every feature must be usable by a volunteer with minimal training on a mobile phone during a busy event.

2. **Offline-first**: Field operations must work without internet. Data syncs when connectivity returns.

3. **Privacy by design**: LGPD compliance is not a feature — it is a constraint on every feature.

4. **Auditability**: Every data access and change must be traceable for compliance and accountability.

5. **Mobile-first, desktop-capable**: Design for the phone screen first; adapt to desktop second.

6. **Incremental delivery**: Ship a working MVP quickly; add capabilities in phases.

7. **Low operational cost**: Use free/open-source tools. Minimize infrastructure complexity. Design for a constrained budget.

---

## Success Criteria

### MVP Success (Phase 1)
- [ ] Beneficiaries can be registered via mobile device
- [ ] Volunteers can record triage and basic attendance
- [ ] Data persists offline and syncs when online
- [ ] Administrators can manage users and roles
- [ ] Basic attendance reports can be generated

### Platform Success (Phase 2-3)
- [ ] Full service workflow (triage → attendance → follow-up → completion)
- [ ] Campaign and event management with team coordination
- [ ] Donation recording and receipt generation
- [ ] LGPD consent capture with digital signature
- [ ] Multi-campus data segregation
- [ ] Export to CSV/spreadsheet
- [ ] Statistical dashboards with charts

### Organizational Success (Ongoing)
- [ ] 80%+ volunteer adoption rate at field events
- [ ] Reduction in data entry errors vs. manual processes
- [ ] Complete audit trail for compliance reviews
- [ ] Management reports generated in < 5 minutes
- [ ] Zero sensitive data exposure incidents

---

## Scope Boundaries

### In Scope
- Person management (unified registry with multi-role)
- Attendance workflows (triage, service, follow-up)
- Campaign and event management
- Donation tracking
- RBAC and audit trail
- Offline-first mobile interface
- Reports and data export
- LGPD consent management
- Multi-campus support

### Out of Scope (for now)
- Financial accounting (ERP integration)
- Payroll management
- Inventory management
- E-commerce or online donation portals
- Integration with government systems
- Automated notification systems (SMS, WhatsApp)
- Real-time chat or messaging
- Video conferencing
- AI-powered analytics or recommendations
