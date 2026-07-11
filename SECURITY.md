# Security Policy

## Supported versions

goboot is pre-1.0 and under active development. Security fixes are applied to the
`main` branch and released in the next tagged version. Until 1.0, only the latest
release line is supported.

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, report them privately using one of:

- GitHub's [private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
  (the **Report a vulnerability** button under the repository's *Security* tab), or
- email to **zombocoder@gmail.com** with the subject line `goboot security`.

Please include:

- a description of the vulnerability and its impact,
- steps to reproduce (a minimal annotated snippet and the resulting generated
  code or diagnostic is ideal),
- affected version or commit, and
- any suggested remediation.

We will acknowledge your report within a few business days, keep you informed of
progress, and credit you in the advisory unless you prefer to remain anonymous.

## Scope and hardening notes

goboot is a build-time code generator plus a small runtime. Areas of particular
security relevance (see specification §50):

- Generated SQL uses driver placeholders and never concatenates untrusted values.
- Generated JSON responses are encoded through the standard library.
- The default error handler withholds internal (5xx) messages from responses.
- Generated HTTP handlers include panic recovery and honor context cancellation.
- Annotation content is treated as untrusted compiler input; parsers are
  fuzz-tested to never panic.
- The compiler does not execute analyzed source and prefers direct Go APIs over
  shell invocation.

Generated source must never embed secret configuration values; provide secrets
through configuration sources at runtime.
