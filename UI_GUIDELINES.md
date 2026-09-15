# UI Guidelines

The platform UIs should feel like one product while respecting each module's workflow. Preserve the established design-system tokens and shared navigation rather than introducing isolated patterns.

- Show the active tenant and workspace when a workflow is scoped.
- Hide actions the user cannot perform and still enforce the same rule on the API.
- Provide loading, empty, error, offline, and permission-denied states.
- Make destructive actions explicit, reversible where possible, and keyboard accessible.
- Support responsive layouts, readable contrast, focus states, labels, and screen-reader names.
- Show data freshness, source, and generated or unverified status for evidence views.
- Avoid exposing personal data in notifications, URLs, or screenshots.

Run the relevant production build and, when available, browser accessibility and workflow checks. See `enterprise/design-system/` and the module frontend READMEs.
