# scribe

A **Razor** component engine for **email + PDF**. Templates are built from component base-classes that use `l10n` + `a11y` (localised and accessible by construction). For **email**, the base-classes hide the awful email-HTML quirks so you write clean components → HTML. For **PDF**, that HTML renders via a headless-browser service (**Playwright**). One component source → email HTML + PDF.

Consumer of the stack; sits on `l10n` + `a11y` (+ `f8n` transitively); can embed `etch` graphics (as PNG).

Design: `./DESIGN.md`. Status: skeleton.
