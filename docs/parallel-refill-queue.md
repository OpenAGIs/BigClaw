# BigClaw v5.3 Go Mainline Refill Queue

This file is the human-readable companion to `docs/parallel-refill-queue.json`.
It records the current Go-mainline cutover backlog slices and the refill order
used by the repo-native local tracker in `local-issues.json`.

Linear issue creation is still blocked by workspace issue limits, but BigClaw no
longer waits on Linear to keep issue execution moving.

## Trigger

- Manual one-shot refill:
  - `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json`
- Continuous refill watcher:
  - `bash scripts/ops/bigclawctl refill --apply --watch --local-issues local-issues.json`
- Optional dashboard refresh after promotion:
  - `bash scripts/ops/bigclawctl refill --apply --watch --local-issues local-issues.json --refresh-url http://127.0.0.1:4000/api/v1/refresh`
- Local issue CLI:
  - `bash scripts/ops/bigclaw-issue list`
  - `bash scripts/ops/bigclaw-issue state BIG-GOM-303 "In Progress"`
- Local dashboard/orchestrator:
  - `bash scripts/ops/bigclaw-symphony`
  - `bash scripts/ops/bigclaw-panel`

## Policy

- Target: keep `2` issues in `In Progress` when issue capacity is available again.
- Target: keep `2` issues in `In Progress` in the local tracker unless a higher
  parallelism cap is explicitly chosen for a branch-safe batch.
- Promote only issues currently in `Backlog` or `Todo`.
- Use the queue order below as the single source of truth for refill priority.
- Every substantive code-bearing update must be committed and pushed to GitHub immediately, with local/remote SHA equality verification after each push.
- Shared mirror bootstrap remains mandatory so multiple Symphony issues reuse one local mirror/seed cache instead of re-downloading the repo.
- `local-issues.json` is the authoritative issue state backend for ongoing work.
- Use `docs/go-mainline-cutover-issue-pack.md` as the detailed project brief behind this queue.

## Repo Validation

- Current mainline expectation:
  - new implementation work lands in `bigclaw-go`
  - Python paths are migration-only unless explicitly marked otherwise
- Current tracker expectation:
  - issue state lives in `local-issues.json`
  - queue promotion is handled by `bigclawctl refill`
- Repo-native cutover plan:
  - `docs/go-mainline-cutover-issue-pack.md`

## Current batch

- Current repo tranche status as of March 24, 2026:
  - active slices: none
  - standby slices: none
  - recently completed slices: `BIG-PAR-361` — Add ClawHost workflow max helper unit coverage; `BIG-PAR-362` — Add ClawHost workflow report edge-case coverage; `BIG-PAR-363` — Add ClawHost rollout report edge-case coverage; `BIG-PAR-364` — Add ClawHost fleet report edge-case coverage; `BIG-PAR-365` — Add product console unit coverage; `BIG-PAR-366` — Add saved-view helper unit coverage; `BIG-PAR-367` — Add dashboard run contract path helper coverage; `BIG-PAR-368` — Broaden direct recovery builder coverage
  - queue status: `queue_runnable=0`, `target_in_progress=2`
  - run `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json --sync-queue-status` to keep queue status, recent batches, and this markdown companion aligned after tracker changes
- Queue drained recovery:
  - if `bigclawctl refill` reports `queue_drained: true`, the queue has no runnable identifiers left in `docs/parallel-refill-queue.json`
  - seed the next `BIG-PAR-*` identifier with `bash scripts/ops/bigclawctl refill seed --local-issues local-issues.json --identifier BIG-PAR-XXX --title "..." --state Todo --recent-batch standby --json`
  - once the next batch exists, run `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json --sync-queue-status` to align queue metadata and this markdown companion with the local tracker state
- Completed slices:
  - `BIG-GOM-301` — Unified domain model and intake contract migration
  - `BIG-GOM-302` — Risk, policy, and approval semantics migration
  - `BIG-GOM-303` — Workflow orchestration and scheduler loop migration
  - `BIG-GOM-304` — Observability, reporting, and weekly operations surface migration
  - `BIG-GOM-305` — Control center, triage, and operations view migration
  - `BIG-GOM-306` — Repo collaboration and lineage surface migration
  - `BIG-GOM-307` — Workflow, bootstrap, and GitHub sync toolchain migration
  - `BIG-GOM-308` — Python deprecation and Go-only mainline switch
  - `BIG-PAR-219` — Expose ahead/behind relation in github-sync status payload
  - `BIG-PAR-220` — Go-first traceability refresh for issue plan evidence pointers
  - `BIG-PAR-221` — Draft parallel validation matrix artifact for local/k8s/ray
  - `BIG-PAR-222` — Add fast compile-check path for frozen legacy Python shims
  - `BIG-PAR-223` — Link planning docs to validation matrix
  - `BIG-PAR-224` — Reconcile tracker and refill queue follow-up state
  - `BIG-PAR-225` — Publish canonical parallel follow-up index
  - `BIG-PAR-226` — Rewire maintained reports to canonical follow-up index
  - `BIG-PAR-227` — Rewire migration plan review notes to follow-up index
  - `BIG-PAR-228` — Rewire per-run bundle READMEs to follow-up index
  - `BIG-PAR-229` — Rename maintained follow-up sections to follow-up index
  - `BIG-PAR-230` — Rename bundle README follow-up sections to follow-up index
  - `BIG-PAR-231` — Restore required follow-up digest references for CI
  - `BIG-PAR-234` — bigclawctl: support --help at root and subcommands
  - `BIG-PAR-235` — cap workflow agent fanout after 429s
  - `BIG-PAR-236` — harden local tracker recovery and serialization
  - `BIG-PAR-237` — emit queue-drained recovery hints in refill output
  - `BIG-PAR-238` — bigclawctl refill: seed queue entries from CLI
  - `BIG-PAR-239` — bigclawctl refill: sync recent_batches metadata from local tracker
  - `BIG-PAR-240` — Document queue seeding and drained-batch recovery workflow
  - `BIG-PAR-241` — Serialize local tracker writes with an explicit lock
  - `BIG-PAR-242` — Sync refill recent-batch metadata from the local tracker
  - `BIG-PAR-243` — Reload local tracker state on each refill fetch
  - `BIG-PAR-244` — Refresh refill queue docs for current local-backend behavior
  - `BIG-PAR-245` — Open PR for tracker and refill hardening branch
  - `BIG-PAR-246` — Refresh PR branch against main
  - `BIG-PAR-247` — bigclawctl refill: sync queue markdown from canonical state
  - `BIG-PAR-248` — Expand SQLite queue reliability proof to 10k tasks
  - `BIG-PAR-249` — Refresh queue reliability references after 10k proof
  - `BIG-PAR-250` — Refresh Go-mainline handoff note for merged cutover state
  - `BIG-PAR-251` — Fix rollback follow-up issue ID drift in gap analysis
  - `BIG-PAR-252` — Add observability follow-up doc regression coverage
  - `BIG-PAR-253` — Add migration and validation follow-up doc regression coverage
  - `BIG-PAR-254` — Add runtime report follow-up ID coverage
  - `BIG-PAR-255` — Align live validation bundle follow-up IDs
  - `BIG-PAR-256` — Align live validation index JSON follow-up metadata
  - `BIG-PAR-257` — Align continuation gate JSON follow-up metadata
  - `BIG-PAR-258` — Align rollback trigger JSON follow-up metadata
  - `BIG-PAR-259` — Align live shadow JSON follow-up metadata
  - `BIG-PAR-260` — Align live shadow bundle follow-up IDs
  - `BIG-PAR-261` — Align migration readiness live-shadow follow-up ID
  - `BIG-PAR-268` — Rewire readiness reports to canonical follow-up index
  - `BIG-PAR-269` — Add canonical validation matrix regression coverage
  - `BIG-PAR-270` — Add live validation summary regression coverage
  - `BIG-PAR-271` — Add broker validation summary regression coverage
  - `BIG-PAR-272` — Add shared queue companion summary regression coverage
  - `BIG-PAR-273` — Add live validation index regression coverage
  - `BIG-PAR-274` — Add shared queue report regression coverage
  - `BIG-PAR-275` — Add observability follow-up regression coverage
  - `BIG-PAR-276` — Add coordination contract-only regression coverage
  - `BIG-PAR-277` — Add live-shadow rollback bundle regression coverage
  - `BIG-PAR-278` — Add production corpus coverage regression surface
  - `BIG-PAR-279` — Add subscriber takeover proof regression surface
  - `BIG-PAR-280` — Add durability rollout review bundle regression surface
  - `BIG-PAR-283` — Add provider live handoff isolation regression coverage
  - `BIG-PAR-282` — Add sequence and retention surface regression coverage
  - `BIG-PAR-284` — Refactor control center response assembly
  - `BIG-PAR-285` — Refactor distributed diagnostics builders
  - `BIG-PAR-286` — Refactor worker runtime RunOnce flow
  - `BIG-PAR-287` — Add ClawHost fleet inventory and control-plane source
  - `BIG-PAR-288` — Add parallel ClawHost rollout planner
  - `BIG-PAR-289` — Add ClawHost skills channels and device approval workflows
  - `BIG-PAR-290` — Add ClawHost provider defaults and tenant policy surfaces
  - `BIG-PAR-291` — Add ClawHost proxy subdomain and admin validation lane
  - `BIG-PAR-292` — Add ClawHost lifecycle recovery and per-bot isolation scorecard
  - `BIG-PAR-293` — Add control-center regression coverage for ClawHost recovery surface
  - `BIG-PAR-294` — Add control-center regression coverage for ClawHost workflow surface
  - `BIG-PAR-295` — Add control-center regression coverage for ClawHost readiness surface
  - `BIG-PAR-296` — Add control-center regression coverage for ClawHost policy surface
  - `BIG-PAR-297` — Add aggregate control-center regression for ClawHost surface bundle
  - `BIG-PAR-298` — Add ClawHost policy report and export surface
  - `BIG-PAR-299` — Scope ClawHost policy surface by team and project
  - `BIG-PAR-300` — Scope ClawHost control-center surfaces by active filters
  - `BIG-PAR-301` — Reuse one ClawHost snapshot across debug status surfaces
  - `BIG-PAR-302` — Add team and project scoping to ClawHost debug surfaces
  - `BIG-PAR-303` — Add team and project scoping to ClawHost fleet surface
  - `BIG-PAR-304` — Harden scoped ClawHost expansion export regressions
  - `BIG-PAR-305` — Add product regression coverage for ClawHost fleet filtering
  - `BIG-PAR-306` — Add product regression coverage for scoped ClawHost recovery scorecards
  - `BIG-PAR-307` — Add product regression coverage for scoped ClawHost workflow lanes
  - `BIG-PAR-308` — Add product regression coverage for scoped ClawHost rollout planners
  - `BIG-PAR-309` — Add product regression coverage for ClawHost workflow review reasons
  - `BIG-PAR-310` — Add product regression coverage for ClawHost workflow review ordering
  - `BIG-PAR-311` — Add product regression coverage for idle ClawHost workflow surfaces
  - `BIG-PAR-312` — Add product regression coverage for ClawHost workflow lane concurrency
  - `BIG-PAR-313` — Add product regression coverage for idle ClawHost workflow lane signals
  - `BIG-PAR-314` — Add product regression coverage for empty ClawHost fleet reports
  - `BIG-PAR-315` — Add product regression coverage for empty ClawHost recovery reports
  - `BIG-PAR-316` — Add product regression coverage for fallback ClawHost rollout reports
  - `BIG-PAR-317` — Add product regression coverage for empty ClawHost workflow lane reports
  - `BIG-PAR-318` — Add product regression coverage for empty ClawHost rollout audits
  - `BIG-PAR-319` — Add product regression coverage for empty ClawHost recovery audits
  - `BIG-PAR-320` — Add product regression coverage for ClawHost fleet compatibility aliases
  - `BIG-PAR-321` — Add product regression coverage for default ClawHost fleet compatibility alias
  - `BIG-PAR-322` — Add product regression coverage for ClawHost workflow parsing defaults
  - `BIG-PAR-323` — Add product regression coverage for ClawHost workflow pairing defaults
  - `BIG-PAR-324` — Add product regression coverage for ClawHost workflow metadata precedence
  - `BIG-PAR-325` — Add product regression coverage for ClawHost workflow lane compatibility alias
  - `BIG-PAR-326` — Expose ClawHost workflow report filters and operational signals
  - `BIG-PAR-327` — Expose ClawHost rollout report filters
  - `BIG-PAR-328` — Expose ClawHost recovery report filters
  - `BIG-PAR-329` — Expose ClawHost policy report filters
  - `BIG-PAR-330` — Expose ClawHost fleet report filters
  - `BIG-PAR-331` — Add filters to ClawHost policy surface payload
  - `BIG-PAR-332` — Add filters to ClawHost rollout surface payload
  - `BIG-PAR-333` — Add filters to ClawHost recovery surface payload
  - `BIG-PAR-334` — Add filters to ClawHost readiness surface payload
  - `BIG-PAR-335` — Thread filters into ClawHost workflow surface payload
  - `BIG-PAR-336` — Add workflow filter assertions to API regressions
  - `BIG-PAR-337` — Add scoped debug filter assertions for remaining ClawHost surfaces
  - `BIG-PAR-338` — Add unscoped bundle filter assertions for remaining ClawHost surfaces
  - `BIG-PAR-339` — Add policy filter assertions to bundle regressions
  - `BIG-PAR-340` — Add policy filter assertions to standalone API regressions
  - `BIG-PAR-341` — Add scoped debug policy filter assertion
  - `BIG-PAR-342` — Pin default ClawHost product filter maps in regressions
  - `BIG-PAR-343` — Pin default fleet filter maps in regressions
  - `BIG-PAR-344` — Add policy report fallback filter regression
  - `BIG-PAR-345` — Pin workflow-lane idle filter maps in regressions
  - `BIG-PAR-346` — Add policy surface empty-state unit coverage
  - `BIG-PAR-347` — ClawHost expansion endpoints preserve default unscoped reports
  - `BIG-PAR-348` — Add ClawHost expansion helper unit coverage
  - `BIG-PAR-349` — Add ClawHost expansion method guard coverage
  - `BIG-PAR-350` — Add ClawHost readiness surface unit coverage
  - `BIG-PAR-351` — Add ClawHost recovery surface unit coverage
  - `BIG-PAR-352` — Add ClawHost rollout surface unit coverage
  - `BIG-PAR-353` — Add ClawHost workflow surface payload unit coverage
  - `BIG-PAR-354` — Add ClawHost parsing helper unit coverage
  - `BIG-PAR-355` — Add ClawHost policy helper unit coverage
  - `BIG-PAR-356` — Add ClawHost fleet helper unit coverage
  - `BIG-PAR-357` — Add ClawHost rollout helper unit coverage
  - `BIG-PAR-358` — Add ClawHost workflow helper unit coverage
  - `BIG-PAR-359` — Add ClawHost recovery report edge-case coverage
  - `BIG-PAR-360` — Extract ClawHost recovery builder for direct inventory coverage
  - `BIG-PAR-361` — Add ClawHost workflow max helper unit coverage
  - `BIG-PAR-362` — Add ClawHost workflow report edge-case coverage
  - `BIG-PAR-363` — Add ClawHost rollout report edge-case coverage
  - `BIG-PAR-364` — Add ClawHost fleet report edge-case coverage
  - `BIG-PAR-365` — Add product console unit coverage
  - `BIG-PAR-366` — Add saved-view helper unit coverage
  - `BIG-PAR-367` — Add dashboard run contract path helper coverage
  - `BIG-PAR-368` — Broaden direct recovery builder coverage
- Historical first runnable batch once issue creation was available:
  - `BIG-GOM-301` — Unified domain model and intake contract migration
  - `BIG-GOM-302` — Risk, policy, and approval semantics migration
  - `BIG-GOM-303` — Workflow orchestration and scheduler loop migration
  - `BIG-GOM-304` — Observability, reporting, and weekly operations surface migration

## Canonical refill order

1. `BIG-GOM-301`
2. `BIG-GOM-302`
3. `BIG-GOM-303`
4. `BIG-GOM-304`
5. `BIG-GOM-305`
6. `BIG-GOM-306`
7. `BIG-GOM-307`
8. `BIG-GOM-308`
9. `BIG-PAR-219`
10. `BIG-PAR-220`
11. `BIG-PAR-221`
12. `BIG-PAR-222`
13. `BIG-PAR-223`
14. `BIG-PAR-224`
15. `BIG-PAR-225`
16. `BIG-PAR-226`
17. `BIG-PAR-227`
18. `BIG-PAR-228`
19. `BIG-PAR-229`
20. `BIG-PAR-230`
21. `BIG-PAR-231`
22. `BIG-PAR-234`
23. `BIG-PAR-235`
24. `BIG-PAR-236`
25. `BIG-PAR-237`
26. `BIG-PAR-238`
27. `BIG-PAR-239`
28. `BIG-PAR-240`
29. `BIG-PAR-241`
30. `BIG-PAR-242`
31. `BIG-PAR-243`
32. `BIG-PAR-244`
33. `BIG-PAR-245`
34. `BIG-PAR-246`
35. `BIG-PAR-247`
36. `BIG-PAR-248`
37. `BIG-PAR-249`
38. `BIG-PAR-250`
39. `BIG-PAR-251`
40. `BIG-PAR-252`
41. `BIG-PAR-253`
42. `BIG-PAR-254`
43. `BIG-PAR-255`
44. `BIG-PAR-256`
45. `BIG-PAR-257`
46. `BIG-PAR-258`
47. `BIG-PAR-259`
48. `BIG-PAR-260`
49. `BIG-PAR-261`
50. `BIG-PAR-268`
51. `BIG-PAR-269`
52. `BIG-PAR-270`
53. `BIG-PAR-271`
54. `BIG-PAR-272`
55. `BIG-PAR-273`
56. `BIG-PAR-274`
57. `BIG-PAR-275`
58. `BIG-PAR-276`
59. `BIG-PAR-277`
60. `BIG-PAR-278`
61. `BIG-PAR-279`
62. `BIG-PAR-280`
63. `BIG-PAR-283`
64. `BIG-PAR-282`
65. `BIG-PAR-284`
66. `BIG-PAR-285`
67. `BIG-PAR-286`
68. `BIG-PAR-287`
69. `BIG-PAR-288`
70. `BIG-PAR-289`
71. `BIG-PAR-290`
72. `BIG-PAR-291`
73. `BIG-PAR-292`
74. `BIG-PAR-293`
75. `BIG-PAR-294`
76. `BIG-PAR-295`
77. `BIG-PAR-296`
78. `BIG-PAR-297`
79. `BIG-PAR-298`
80. `BIG-PAR-299`
81. `BIG-PAR-300`
82. `BIG-PAR-301`
83. `BIG-PAR-302`
84. `BIG-PAR-303`
85. `BIG-PAR-304`
86. `BIG-PAR-305`
87. `BIG-PAR-306`
88. `BIG-PAR-307`
89. `BIG-PAR-308`
90. `BIG-PAR-309`
91. `BIG-PAR-310`
92. `BIG-PAR-311`
93. `BIG-PAR-312`
94. `BIG-PAR-313`
95. `BIG-PAR-314`
96. `BIG-PAR-315`
97. `BIG-PAR-316`
98. `BIG-PAR-317`
99. `BIG-PAR-318`
100. `BIG-PAR-319`
101. `BIG-PAR-320`
102. `BIG-PAR-321`
103. `BIG-PAR-322`
104. `BIG-PAR-323`
105. `BIG-PAR-324`
106. `BIG-PAR-325`
107. `BIG-PAR-326`
108. `BIG-PAR-327`
109. `BIG-PAR-328`
110. `BIG-PAR-329`
111. `BIG-PAR-330`
112. `BIG-PAR-331`
113. `BIG-PAR-332`
114. `BIG-PAR-333`
115. `BIG-PAR-334`
116. `BIG-PAR-335`
117. `BIG-PAR-336`
118. `BIG-PAR-337`
119. `BIG-PAR-338`
120. `BIG-PAR-339`
121. `BIG-PAR-340`
122. `BIG-PAR-341`
123. `BIG-PAR-342`
124. `BIG-PAR-343`
125. `BIG-PAR-344`
126. `BIG-PAR-345`
127. `BIG-PAR-346`
128. `BIG-PAR-347`
129. `BIG-PAR-348`
130. `BIG-PAR-349`
131. `BIG-PAR-350`
132. `BIG-PAR-351`
133. `BIG-PAR-352`
134. `BIG-PAR-353`
135. `BIG-PAR-354`
136. `BIG-PAR-355`
137. `BIG-PAR-356`
138. `BIG-PAR-357`
139. `BIG-PAR-358`
140. `BIG-PAR-359`
141. `BIG-PAR-360`
142. `BIG-PAR-361`
143. `BIG-PAR-362`
144. `BIG-PAR-363`
145. `BIG-PAR-364`
146. `BIG-PAR-365`
147. `BIG-PAR-366`
148. `BIG-PAR-367`
149. `BIG-PAR-368`
