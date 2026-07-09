# scribe

A **TS/Vue** component engine for **email + PDF**. Templates are clean components that use `l10n` + `a11y` (localised and accessible by construction) and can embed `etch` graphics. For **email**, the components compile to bulletproof email HTML (hiding the email-HTML quirks); for **PDF**, that HTML renders via a headless-browser service (**Playwright**). One component source → email HTML + PDF.

Consumer of the stack; sits on `l10n` + `a11y` (+ `f8n` transitively). Same Vue-component model as `etch`, so one component model spans UI, email, and PDF.

Design: `./DESIGN.md`. Status: skeleton (templating engine open — MJML / vue-email / Vue-SSR).
