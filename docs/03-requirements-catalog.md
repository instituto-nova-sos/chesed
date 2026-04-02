# 03 - Requirements Catalog

## Functional Requirements

### FR-01: Unified Person Registry

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-01 | The system must allow a unique person registry | Must | Central entity; all roles reference the same person |
| RF-02 | The system must allow recording basic personal data (name, birth date, phone, address, CPF or equivalent document) | Must | Address should include street, number, neighborhood, city, state, zip code |
| RF-03 | The system must prevent duplicate person records | Must | Deduplication by CPF/document + fuzzy name matching |
| RF-04 | The system must allow updating registration data | Must | With audit trail |
| RF-05 | The system must keep a history of registration changes | Must | Audit log of field-level changes |
| RF-06 | The system must allow attaching documents to a person record | Should | File upload to object storage; types: ID, proof of residence, medical records |
| RF-07 | The system must allow registering signed consents and terms | Must | Digital signature capture on mobile; LGPD requirement |
| RF-08 | The system must allow registering image usage consent | Must | Separate consent type; revocable |

**Additional requirements identified:**
| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-02a | The system must support international document types (CPF, SSN, EU ID) | Should | For multi-region operation |
| RF-02b | The system must allow recording email address | Should | For communication and password recovery |
| RF-03a | The system should suggest potential duplicates during registration | Should | Fuzzy matching on name + birth date when CPF is not available |

### FR-02: Person Role Management

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-09 | A person can be associated with one or more business roles | Must | Multi-role design |
| RF-10 | Supported roles: volunteer, assisted person, health professional, administrator | Must | Extensible role list |
| RF-11 | The same person may simultaneously be assisted and volunteer | Must | Common in community organizations |
| RF-12 | The system must allow activating or deactivating roles | Must | Soft-delete; roles are never physically removed |
| RF-13 | The system must keep role history | Must | When activated/deactivated, by whom |

**Ambiguity resolved:**
- RF-10 lists "health professional" but the broader system needs other professional types (legal, social worker, nutritionist). The role list should be: **volunteer, assisted_person, professional, coordinator, administrator**. Professional type is a separate attribute.

### FR-03: System User Management

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-14 | The system must allow creation of system users associated with a registered person | Must | Not all persons are system users |
| RF-15 | The system must allow authentication via email and password | Must | Email as login identifier |
| RF-16 | The system must allow password recovery | Must | Via email with secure token |
| RF-17 | The system must implement RBAC | Must | Role-based access control with granular permissions |
| RF-18 | Supported access profiles: administrator, coordinator, professional, secretary, volunteer | Must | Maps to permission sets |

### FR-04: Assisted Person Extended Profile

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-19 | The system must allow recording additional information for assisted persons | Must | Social vulnerability data, family composition, income range |
| RF-20 | The system must allow registering relevant social information for follow-up | Must | Free-text and structured fields |
| RF-21 | The system must keep the history of the person's relationship as assisted | Must | Timeline of all interactions |

**Additional requirements identified:**
| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-19a | The system should allow recording family/household grouping | Should | Link related beneficiaries |
| RF-19b | The system should allow recording referral source | Should | How the person learned about the organization |

### FR-05: Triage

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-22 | The system must allow recording initial triage | Must | First contact assessment |
| RF-23 | The system must allow recording the main complaint or reason for service | Must | Free-text + categorized |
| RF-24 | The system must allow recording requested or necessary services | Must | Multiple services per triage |
| RF-25 | The system must allow selecting responsible teams | Should | Team assignment for routing |
| RF-26 | The system must allow recording date, place, and campaign | Must | Context of the triage event |

### FR-06: Service Attendance Records

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-27 | The system must allow recording services performed by team or professional | Must | Who did what |
| RF-28 | The system must allow recording services from different areas (legal, medical, nutritional, physiotherapy, social, etc.) | Must | Configurable service type catalog |
| RF-29 | The system must allow recording observations and recommendations | Must | Free-text notes per attendance |
| RF-30 | The system must allow attaching documents or exams | Should | File upload linked to attendance record |
| RF-31 | The system must keep the full attendance history of a person | Must | Chronological timeline |

### FR-07: Service Workflow

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-32 | The system must implement a configurable service workflow | Must | State machine pattern |
| RF-33 | The system must support states: triage, attendance, follow-up, completed | Must | Minimum state set; extensible |
| RF-34 | The system must record workflow state transition history | Must | Who changed, when, from what to what |
| RF-35 | The system must allow reopening attendances for follow-up | Must | Completed → follow-up transition |

**Additional requirement identified:**
| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-35a | The system should allow cancelling an attendance with a reason | Should | For error correction |

### FR-08: Campaign and Event Management

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-36 | The system must allow registering campaigns or social actions | Must | Planned events with metadata |
| RF-37 | The system must allow associating attendances with campaigns or events | Must | Many-to-one relationship |
| RF-38 | The system must allow recording participating teams for each campaign | Should | Team composition per event |
| RF-39 | The system must allow recording place, date, and responsible people | Must | Logistics tracking |

### FR-09: Reports and Statistics

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-40 | The system must generate attendance reports by period | Must | Date range filter |
| RF-41 | The system must generate reports by attendance type | Must | Service category breakdown |
| RF-42 | The system must generate reports by team | Should | Professional/team performance |
| RF-43 | The system must generate reports by campaign or event | Must | Per-campaign metrics |
| RF-44 | The system must allow exporting reports as CSV or spreadsheet | Must | Data portability |
| RF-45 | The system must generate statistical charts | Should | Visual dashboard |

### FR-10: Offline Operation

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-46 | The system must allow offline data entry on mobile devices | Must | Core architectural requirement |
| RF-47 | The system must store data locally until synchronization | Must | IndexedDB or equivalent |
| RF-48 | The system must automatically synchronize when connectivity is restored | Must | Background sync |
| RF-49 | The system must handle synchronization conflicts | Must | Conflict resolution strategy required |

### FR-11: Audit and Traceability

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-50 | The system must log access to data | Must | Read audit for sensitive data |
| RF-51 | The system must record history of record changes | Must | Field-level change tracking |
| RF-52 | The system must record which user viewed or changed sensitive data | Must | User attribution |
| RF-53 | The system must allow audit consultation for compliance purposes | Must | Admin-facing audit viewer |

### FR-12: Donations and Accountability

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-54 | The system must allow registering financial donations and item donations | Must | Amount + type (money, goods, services) |
| RF-55 | The system must allow issuing and downloading proof of donation value | Should | PDF receipt generation |
| RF-56 | The system must allow linking a donation to a campaign or event | Should | Donation attribution |

### FR-13: Consent and LGPD

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-57 | The system must allow issuing and digitally signing consent forms on mobile devices | Must | Touch signature capture |
| RF-58 | The system must allow recording consent revocation and logical deletion of sensitive data upon request | Must | Right to erasure; soft-delete with anonymization |

**Additional requirements identified:**
| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RF-58a | The system must inform the purpose of data collection at the time of consent | Must | LGPD Art. 7 |
| RF-58b | The system must maintain a consent registry with timestamp, version, and scope | Must | Audit trail for consent |
| RF-58c | The system must allow granular consent (e.g., data processing vs. image usage) | Should | Separate consent types |

---

## Non-Functional Requirements

### NFR-01: Security and Privacy

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RNF-01 | The system must comply with LGPD | Must | Foundational constraint |
| RNF-02 | The system must consider data protection standards for Brazil, USA, and Europe | Should | LGPD, CCPA awareness, GDPR awareness |
| RNF-03 | The system must enforce TLS encryption in transit | Must | HTTPS everywhere |
| RNF-04 | The system must enforce encryption at rest, including local device storage | Must | Database encryption + IndexedDB encryption |
| RNF-05 | The system must support audit consultation for compliance | Must | Covered by FR-11 |
| RNF-06 | The system must implement RBAC | Must | Covered by FR-03 |
| RNF-07 | The system must allow data segregation by campus/region | Must | Tenant-like isolation |

### NFR-02: Performance and Scalability

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RNF-08 | The system must support at least 100 concurrent users | Must | Modest requirement; Go handles this easily |
| RNF-09 | The system must provide efficient synchronization for multiple offline devices | Must | Batch sync with minimal bandwidth |
| RNF-10 | The system must support growth in attendance volume over years | Must | Indexing and archival strategy |

### NFR-03: Usability

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RNF-11 | The system must be simple for volunteers with low technical experience | Must | Maximum 3 taps to start recording |
| RNF-12 | The system must support fast operation during high-volume events | Must | Optimized forms, autocomplete, quick-add |
| RNF-13 | The system must provide a responsive interface for mobile and desktop | Must | Mobile-first CSS framework |

### NFR-04: Infrastructure

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RNF-14 | The system must support multi-region operation (Brazil, USA, Europe) | Should | Phase 3+ consideration |
| RNF-15 | The system must allow data segregation by church campus | Must | Campus as a first-class entity |
| RNF-16 | The system must provide automatic backups and disaster recovery | Must | Automated daily backups with tested restore |

### NFR-05: Architecture

| ID | Requirement | Priority | Notes |
|----|------------|----------|-------|
| RNF-17 | The system must follow an API-first architecture | Must | Backend serves JSON; frontend consumes it |
| RNF-18 | The frontend should be implemented as an offline-first PWA | Must | Service Worker + IndexedDB |
| RNF-19 | The UI/UX should follow a mobile-first model | Must | Design for small screens first |
| RNF-20 | The system must provide a secure API for integrations | Should | WordPress portal, future integrations |

---

## Requirements Summary

| Category | Must | Should | Total |
|----------|------|--------|-------|
| Functional | 42 | 16 | 58 |
| Non-Functional | 15 | 5 | 20 |
| **Total** | **57** | **21** | **78** |

---

## Identified Ambiguities and Decisions Needed

1. **Document types for international operation**: What identification documents are accepted in each region? (CPF in Brazil, SSN in USA, national ID in Europe?)
2. **Service type catalog**: Is the list of service types fixed or configurable by administrators?
3. **Workflow customization**: Can different campuses have different workflow states, or is the workflow universal?
4. **Consent form templates**: Who designs the consent form content? Is it managed within the system or uploaded as templates?
5. **Donation receipt format**: Are there legal requirements for the receipt format in each country of operation?
6. **Data retention policy**: How long should records be retained? What is the archival strategy?
7. **Image usage consent scope**: Does image consent cover all media, or are there granular options (social media, print, internal only)?
