---
"@effect/tsgo": patch
---

Stop `experimentalApiUsage` and `unstableApiUsage` from causing TS2589 and follow-on type errors in generic calls such as `Match.when({ kind: "A0" }, ...)` over a wide union. When a generic call infers a property back to the object literal, the rules now read the type parameter's constraint from the signature the checker already resolved, instead of resolving the call again with inference blocked. As a result, constraint properties inside inferred generator functions, array constraints, and rest-parameter constraints are now reported too, for example `generic([{ test: 42 }])` when `T extends Reportable[]` marks `test` as experimental.
