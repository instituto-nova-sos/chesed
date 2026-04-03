# 02 - Problem Statement

## Current Operational Pain Points

### 1. Fragmented Data Management
Beneficiary information is scattered across spreadsheets, paper forms, and informal records. There is no single source of truth. When a person returns for follow-up, volunteers must search through multiple records or rely on memory to find prior history.

**Impact**: Duplicate records, incomplete histories, inconsistent data quality, wasted time during high-volume events.

### 2. No Structured Service History
Service attendance is recorded in ad-hoc ways (if at all). There is no systematic triage-to-follow-up workflow. Professionals cannot see what services a beneficiary has already received, what was recommended, or what follow-up is pending.

**Impact**: Missed follow-ups, repeated assessments, inability to measure service effectiveness, poor continuity of care.

### 3. Limited Reporting Capability
Without centralized data, generating reports requires manual compilation. Leadership cannot easily answer questions like: "How many people did we serve this quarter?", "Which services are most in demand?", or "How effective was our last campaign?"

**Impact**: Weak social impact measurement, difficulty in donor accountability, inability to make data-driven decisions about resource allocation.

### 4. Data Security and Privacy Risks
Personal and sensitive data (health records, social vulnerabilities, CPF numbers) is handled without systematic access controls, encryption, or consent management. Brazil's LGPD (Lei Geral de Proteção de Dados) requires formal consent, purpose limitation, and data subject rights.

**Impact**: Legal risk, reputational risk, ethical responsibility to vulnerable populations who trust the organization with their data.

### 5. No Support for Field Operations
Social action events happen in community spaces, churches, and public areas — often with poor or no internet connectivity. Currently, there is no digital solution for offline data collection during these events.

**Impact**: Paper forms that must be manually transcribed later, data entry errors, delays in data availability, lost forms.

### 6. Volunteer Onboarding Friction
Volunteers have varying levels of technical skill. The previous system (Django Admin) required training and was not intuitive for non-technical users, especially on mobile devices.

**Impact**: Resistance to digital tools, continued reliance on paper, inconsistent data entry, high training overhead for each event.

### 7. No Campaign or Event Coordination
There is no system to plan campaigns, assign teams, track participation, or associate service records with specific events. Campaign logistics are managed informally.

**Impact**: Difficulty coordinating volunteers, inability to track campaign effectiveness, no historical record of what happened at each event.

### 8. No Donation Tracking or Accountability
Financial and in-kind donations are not systematically recorded. Donors cannot receive formal receipts. The organization cannot demonstrate financial transparency.

**Impact**: Missed tax benefits for donors, difficulty reporting to stakeholders, lack of financial transparency.

---

## Why This System Matters

### For Beneficiaries
People in vulnerable situations deserve continuity of care. When a single mother visits a social action event for legal aid and returns three months later for nutritional support, the system should remember her. It should know what services she has received, what was recommended, and what follow-up is pending. Dignity requires that she does not have to re-explain her story every time.

### For Volunteers
Volunteers give their time freely. They should not spend that time fighting with confusing software or filling out redundant forms. The system should make their work easier, not harder. Simple triage flows on a mobile phone, even without internet, should be the standard.

### For the Organization
Instituto Nova SOS needs to demonstrate social impact to sustain its mission. Accurate data enables better resource allocation, stronger donor relationships, and evidence-based program design. Without data, the organization operates on intuition.

### For Compliance
LGPD is not optional. The organization handles sensitive personal data — health information, social vulnerabilities, identification documents — of people who may not fully understand how their data is used. Proper consent management, access controls, and audit trails are ethical obligations, not just legal requirements.

---

## Risks of Not Solving This

| Risk | Likelihood | Impact |
|------|-----------|--------|
| LGPD violation leading to fines or sanctions | Medium | High |
| Loss of beneficiary data due to ad-hoc storage | High | High |
| Inability to demonstrate impact to donors | High | Medium |
| Volunteer burnout from inefficient tools | Medium | Medium |
| Duplicate or conflicting records causing errors in care | High | High |
| Missed follow-ups for people with urgent needs | Medium | High |
| Inability to scale operations to new campuses/regions | High | Medium |
| Reputational damage from data breach or mishandling | Low | Very High |

---

## Opportunity

A well-designed, mobile-first, offline-capable system would:

1. **Save time**: Eliminate paper-to-digital transcription; reduce duplicate data entry
2. **Improve care**: Enable continuity of service through complete beneficiary history
3. **Enable scale**: Allow the organization to expand to new regions with consistent processes
4. **Ensure compliance**: Build LGPD controls into the foundation, not as an afterthought
5. **Demonstrate impact**: Generate reports that show donors, leadership, and the community what the organization accomplishes
6. **Empower volunteers**: Give them a tool they can use confidently in the field
7. **Reduce risk**: Centralized, access-controlled data reduces the attack surface for breaches

The cost of building this system is an investment in the organization's operational maturity. The cost of not building it is continued fragmentation, risk, and missed opportunities to serve more people effectively.
