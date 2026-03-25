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

- Current repo tranche status as of March 25, 2026:
  - active slices: none
  - standby slices: none
  - recently completed slices: `BIG-PAR-402` — Disable HTML escaping in bigclawctl JSON output; `BIG-PAR-403` — Disable HTML escaping in direct bigclawctl JSON encoders; `BIG-PAR-404` — Add workspace validate JSON no-escape regression; `BIG-PAR-405` — Add refill JSON no-escape regression; `BIG-PAR-406` — Add workspace bootstrap JSON no-escape regression; `BIG-PAR-407` — Add workspace cleanup JSON no-escape regression; `BIG-PAR-408` — Add legacy-python JSON no-escape regression; `BIG-PAR-409` — Add github-sync install JSON no-escape regression
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
  - `BIG-PAR-293` — Refresh ClawHost control-plane branch against main
  - `BIG-PAR-294` — Publish ClawHost control-plane review index
  - `BIG-PAR-295` — Add ClawHost surface coexistence regression coverage
  - `BIG-PAR-296` — Add ClawHost export coexistence regression coverage
  - `BIG-PAR-297` — Add ClawHost workflow actor-header regression coverage
  - `BIG-PAR-298` — Add ClawHost endpoint method-guard regression coverage
  - `BIG-PAR-299` — Add ClawHost scope-filter normalization regression coverage
  - `BIG-PAR-300` — Add ClawHost helper contract regression coverage
  - `BIG-PAR-301` — Add ClawHost export header regression coverage
  - `BIG-PAR-302` — Add ClawHost blank-actor fallback regression coverage
  - `BIG-PAR-303` — Add ClawHost empty-actor export omission coverage
  - `BIG-PAR-304` — Add ClawHost partial export-filter regression coverage
  - `BIG-PAR-305` — Add ClawHost blank scope-filter normalization coverage
  - `BIG-PAR-306` — Add workflow endpoint header-actor export URL coverage
  - `BIG-PAR-307` — Add rollout planner actor-omission export URL coverage
  - `BIG-PAR-308` — Add fleet export URL filter-omission coverage
  - `BIG-PAR-309` — Add rollout planner scope-normalization coverage
  - `BIG-PAR-310` — Add direct workflow export header-fallback coverage
  - `BIG-PAR-311` — Add workflow export filename fallback coverage
  - `BIG-PAR-312` — Encode scoped ClawHost saved-view routes
  - `BIG-PAR-313` — Sanitize scoped ClawHost saved-view IDs
  - `BIG-PAR-314` — Encode scoped export URLs for saved views and weekly reports
  - `BIG-PAR-315` — Sanitize attachment filenames for run report exports
  - `BIG-PAR-316` — Add distributed export filename sanitization coverage
  - `BIG-PAR-317` — Add run report fallback filename sanitization coverage
  - `BIG-PAR-318` — Add distributed export fallback filename sanitization coverage
  - `BIG-PAR-319` — Add distributed export project-scope filename sanitization coverage
  - `BIG-PAR-320` — Add distributed export task-scope filename sanitization coverage
  - `BIG-PAR-321` — Add distributed export project-fallback filename sanitization coverage
  - `BIG-PAR-322` — Add distributed export task-fallback filename sanitization coverage
  - `BIG-PAR-323` — Add direct sanitizeReportName helper coverage
  - `BIG-PAR-324` — Add distributed export filename-scope precedence coverage
  - `BIG-PAR-325` — Fix distributed export filename fallback precedence
  - `BIG-PAR-326` — Add distributed export task fallback-after-team coverage
  - `BIG-PAR-327` — Add firstMeaningfulReportName helper coverage
  - `BIG-PAR-328` — Add weeklyExportURL helper coverage
  - `BIG-PAR-329` — Normalize distributedExportURL helper filters
  - `BIG-PAR-330` — Encode controlActionAuditURL query parameters
  - `BIG-PAR-331` — Add savedViewScopeToken helper coverage
  - `BIG-PAR-332` — Add buildSavedViewRoute blank-scope coverage
  - `BIG-PAR-333` — Add viewScopeSuffix punctuation-fallback coverage
  - `BIG-PAR-334` — Add normalizedViewOwner helper coverage
  - `BIG-PAR-335` — Add digestRecipients helper coverage
  - `BIG-PAR-336` — Add saved view metric helper coverage
  - `BIG-PAR-338` — Add saved view rendering helper coverage
  - `BIG-PAR-337` — Add saved view scope and duplicate helper coverage
  - `BIG-PAR-340` — Add saved view report empty-state coverage
  - `BIG-PAR-339` — Add saved view audit readiness edge coverage
  - `BIG-PAR-342` — Add saved view populated report fallback coverage
  - `BIG-PAR-343` — Add unscoped saved view catalog baseline coverage
  - `BIG-PAR-345` — Add valid saved view audit clean-path coverage
  - `BIG-PAR-346` — Add project-scoped saved view catalog coverage
  - `BIG-PAR-348` — Add saved view catalog actor fallback coverage
  - `BIG-PAR-347` — Add team-scoped saved view catalog coverage
  - `BIG-PAR-349` — Add premium-only saved view catalog coverage
  - `BIG-PAR-352` — Add saved view readiness rounding coverage
  - `BIG-PAR-351` — Add saved view catalog ordering coverage
  - `BIG-PAR-353` — Add saved view baseline field coverage
  - `BIG-PAR-355` — Add invalid-visibility audit coverage for saved view catalogs
  - `BIG-PAR-356` — Add direct missing-recipient audit assertions for saved view digests
  - `BIG-PAR-358` — Add ClawHost fleet helper regression coverage
  - `BIG-PAR-357` — Add ClawHost fleet inventory alias wrapper coverage
  - `BIG-PAR-360` — Add console helper and design-system coverage
  - `BIG-PAR-362` — Add ClawHost workflow helper threshold coverage
  - `BIG-PAR-364` — Add ClawHost rollout audit edge coverage
  - `BIG-PAR-365` — Add dashboard contract path traversal coverage
  - `BIG-PAR-367` — Add dashboard contract non-map path traversal coverage
  - `BIG-PAR-369` — Add ClawHost workflow empty-audit coverage
  - `BIG-PAR-370` — Add ClawHost workflow report empty-state coverage
  - `BIG-PAR-371` — Add rollout sorted-values blank-skip coverage
  - `BIG-PAR-372` — Refresh ClawHost review index for merged control-plane surfaces
  - `BIG-PAR-373` — Add ClawHost merged review-doc regression coverage
  - `BIG-PAR-374` — Fix refill dry-run queue_status_synced drift
  - `BIG-PAR-375` — Add refill dry-run drift preview regression
  - `BIG-PAR-376` — Fix refill dry-run recent_batches_synced drift
  - `BIG-PAR-377` — Fix refill apply write flags for queue metadata sync
  - `BIG-PAR-378` — Fix refill dry-run recent batch update mutation
  - `BIG-PAR-379` — Report refill markdown companion writes
  - `BIG-PAR-380` — Document refill markdown_written payload
  - `BIG-PAR-381` — Preview refill markdown writes across promotions
  - `BIG-PAR-382` — Normalize refill payload paths to absolute
  - `BIG-PAR-383` — Add markdown preview helper regression coverage
  - `BIG-PAR-384` — Add queue clone isolation regression
  - `BIG-PAR-386` — Normalize refill active and recent-batch state detection
  - `BIG-PAR-385` — Normalize local tracker state filters for refill commands
  - `BIG-PAR-387` — Normalize queue status sync equivalence
  - `BIG-PAR-388` — Normalize seed and ensure state equivalence
  - `BIG-PAR-390` — Normalize local store state updates
  - `BIG-PAR-391` — Parameterize refill active-state helpers
  - `BIG-PAR-393` — Stabilize normalized refill fetch state lists
  - `BIG-PAR-394` — Canonicalize built-in refill fetch state names
  - `BIG-PAR-395` — Report final synced refill queue state after apply
  - `BIG-PAR-396` — Update existing local issue metadata during ensure
  - `BIG-PAR-397` — Update existing local issue metadata during refill seed
  - `BIG-PAR-398` — Ignore equivalent state spellings in local-issues set-state
  - `BIG-PAR-399` — Ignore equivalent state spellings in local issue store updates
  - `BIG-PAR-400` — Canonicalize equivalent state spellings when creating local issues
  - `BIG-PAR-401` — Canonicalize equivalent queue state spellings during refill seed
  - `BIG-PAR-402` — Disable HTML escaping in bigclawctl JSON output
  - `BIG-PAR-403` — Disable HTML escaping in direct bigclawctl JSON encoders
  - `BIG-PAR-404` — Add workspace validate JSON no-escape regression
  - `BIG-PAR-405` — Add refill JSON no-escape regression
  - `BIG-PAR-406` — Add workspace bootstrap JSON no-escape regression
  - `BIG-PAR-407` — Add workspace cleanup JSON no-escape regression
  - `BIG-PAR-408` — Add legacy-python JSON no-escape regression
  - `BIG-PAR-409` — Add github-sync install JSON no-escape regression
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
118. `BIG-PAR-338`
119. `BIG-PAR-337`
120. `BIG-PAR-340`
121. `BIG-PAR-339`
122. `BIG-PAR-342`
123. `BIG-PAR-343`
124. `BIG-PAR-345`
125. `BIG-PAR-346`
126. `BIG-PAR-348`
127. `BIG-PAR-347`
128. `BIG-PAR-349`
129. `BIG-PAR-352`
130. `BIG-PAR-351`
131. `BIG-PAR-353`
132. `BIG-PAR-355`
133. `BIG-PAR-356`
134. `BIG-PAR-358`
135. `BIG-PAR-357`
136. `BIG-PAR-360`
137. `BIG-PAR-362`
138. `BIG-PAR-364`
139. `BIG-PAR-365`
140. `BIG-PAR-367`
141. `BIG-PAR-369`
142. `BIG-PAR-370`
143. `BIG-PAR-371`
144. `BIG-PAR-372`
145. `BIG-PAR-373`
146. `BIG-PAR-374`
147. `BIG-PAR-375`
148. `BIG-PAR-376`
149. `BIG-PAR-377`
150. `BIG-PAR-378`
151. `BIG-PAR-379`
152. `BIG-PAR-380`
153. `BIG-PAR-381`
154. `BIG-PAR-382`
155. `BIG-PAR-383`
156. `BIG-PAR-384`
157. `BIG-PAR-386`
158. `BIG-PAR-385`
159. `BIG-PAR-387`
160. `BIG-PAR-388`
161. `BIG-PAR-390`
162. `BIG-PAR-391`
163. `BIG-PAR-393`
164. `BIG-PAR-394`
165. `BIG-PAR-395`
166. `BIG-PAR-396`
167. `BIG-PAR-397`
168. `BIG-PAR-398`
169. `BIG-PAR-399`
170. `BIG-PAR-400`
171. `BIG-PAR-401`
172. `BIG-PAR-402`
173. `BIG-PAR-403`
174. `BIG-PAR-404`
175. `BIG-PAR-405`
176. `BIG-PAR-406`
177. `BIG-PAR-407`
178. `BIG-PAR-408`
179. `BIG-PAR-409`
