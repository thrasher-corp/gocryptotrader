# Security Policy

GoCryptoTrader handles exchange API credentials and can place live orders with real funds. We take
security reports seriously and appreciate the effort involved in finding and disclosing them
responsibly.

## Supported Versions

GoCryptoTrader is developed as a rolling release. Only the latest commit on `master` is supported;
fixes are not backported to older commits, tags or forks. Please confirm an issue still reproduces
against current `master` before reporting it.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, pull requests or
Slack.**

Report privately using either of the following:

- **GitHub Security Advisories** (preferred) — go to the
  [Security tab](https://github.com/thrasher-corp/gocryptotrader/security/advisories/new), click
  **Report a vulnerability**, and include as much of the detail below as you can.
- **Email** — <contact@thrasher.io>.

### What to include

- The type of issue and the component affected (exchange wrapper, engine, RPC server, config, etc.).
- The commit hash you tested against.
- Step-by-step reproduction instructions, ideally as a failing Go test or minimal program.
- Impact: what an attacker gains, and what access they need to get it.
- Any proof-of-concept code, logs or configuration needed to reproduce, with credentials redacted.

### What to expect

- **Acknowledgement:** within 5 business days.
- **Assessment:** we will confirm the issue, determine severity and let you know our intended fix
  and timeline.
- **Disclosure:** we aim to release a fix within 90 days of confirmation. We will coordinate public
  disclosure with you and credit you in the advisory unless you prefer to remain anonymous.

Please give us reasonable time to release a fix before disclosing publicly.

## Scope

Vulnerabilities in this repository's code are in scope. Areas of particular interest:

- Leakage of API keys, secrets or client IDs — including via logs, error messages, RPC responses or
  crash output.
- Weaknesses in configuration encryption or credential storage (see [config](/config/README.md)).
- Flaws in exchange request signing or authentication.
- Authentication, authorisation or input handling flaws in the gRPC/REST server (see
  [gctrpc](/gctrpc/README.md)) or the websocket server.
- Sandbox escapes or unintended host access from [gctscript](/gctscript/README.md).
- Parsing or state-handling flaws that let untrusted exchange responses cause memory corruption,
  unbounded resource use or incorrect order placement.
- Dependency vulnerabilities that are actually reachable from GoCryptoTrader code.

### Out of scope

- Financial loss from trading strategies, market conditions or user configuration.
- Vulnerabilities in exchange APIs themselves — report those to the exchange.
- Issues that require an already-compromised host, or physical or root access to the machine running
  the bot.
- Running the RPC server with authentication disabled, weak credentials or bound to a public
  interface where the documentation advises otherwise.
- Missing hardening that has no demonstrated impact, or automated scanner output without a working
  proof of concept.
- Denial of service caused solely by exhausting exchange rate limits.

## Operational Guidance

GoCryptoTrader is a framework, not a hosted service, and its security depends heavily on how it is
deployed. When running it:

- Never commit `config.json` — it may contain API credentials. Use the built-in configuration
  encryption.
- Scope exchange API keys to the minimum permissions needed and disable withdrawal permissions
  unless you require them.
- Restrict the RPC server to a trusted network and change the default credentials.
- Treat logs as sensitive; enable debug logging only when needed and review output before sharing it
  in issues.
