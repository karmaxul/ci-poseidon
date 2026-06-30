# Security Policy

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Report security vulnerabilities privately by emailing [security@healchain.org](mailto:security@healchain.org).

Include as much of the following as possible:

- Type of issue (e.g. smart contract exploit, API authentication bypass, data integrity flaw)
- The affected component (smart contract, oracle, API, SDK, frontend)
- Step-by-step instructions to reproduce
- Proof-of-concept or exploit code if available
- Impact assessment — what an attacker could achieve

We will acknowledge receipt within 48 hours and aim to provide a full response within 7 days.

---

## Scope

### In Scope

- **HealChainStorage smart contracts** — all deployed mainnet and testnet versions
- **Public API** (`api.healchain.org`) — authentication, authorization, data integrity
- **JavaScript SDK** (`@healchain/sdk`) — any issue that could compromise data or credentials
- **Python SDK** (`healchain-sdk`) — same as above
- **Frontend** (`healchain.org`) — XSS, CSRF, credential exposure
- **ci-sha4096 and ci-poseidon** — cryptographic weaknesses or collision vulnerabilities

### Out of Scope

- Theoretical vulnerabilities without proof of concept
- Issues requiring physical access to infrastructure
- Social engineering attacks
- Denial of service via API rate limits (report anyway if severe)
- Issues in third-party dependencies (report to the dependency maintainer)

---

## Supported Versions

Security patches are applied to the current production deployment. We do not backport fixes to older contract versions — if a critical vulnerability is found in a deployed contract, we will redeploy and migrate.

| Component | Supported |
|---|---|
| HealChainStorage v6 (current) | ✅ |
| HealChainStorage v5 and earlier | ⚠️ Read-only / legacy only |
| `@healchain/sdk` latest | ✅ |
| `healchain-sdk` latest | ✅ |

---

## Disclosure Policy

We follow coordinated disclosure. We ask that you:

1. Give us reasonable time to investigate and fix before public disclosure
2. Make a good-faith effort to avoid accessing or modifying data that isn't yours
3. Not perform actions that could harm HealChain users or degrade service availability

In return, we will:

1. Acknowledge your report promptly
2. Keep you informed of our progress
3. Credit you in our disclosure (unless you prefer anonymity)
4. Not pursue legal action against researchers acting in good faith

---

## Contact

**Security:** [security@healchain.org](mailto:security@healchain.org)
**General:** [hello@healchain.org](mailto:hello@healchain.org)

*HealChain is built by Harmony Worldwide.*
