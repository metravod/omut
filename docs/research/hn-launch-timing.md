# Show HN posting time: launch planning for omut

Exploratory analysis, 2026-11-05. Practical recommendation: post in the Moscow
evening, around 19:00–21:00, when the author can remain available for discussion.
The data do not establish a uniquely best weekday or exact hour.

## Data and outcome

- The maintainer's read-only SQLite archive contains 2,390,018 observations of
  HN's top-200 list, covering 25,864 distinct posts, from 2026-06-03 11:25 UTC to
  2026-10-12 17:30 UTC, usually sampled every 15 minutes. It contains 1,760 posts
  classified as Show HN by their observed title or stored kind.
- A denominator was fetched separately from the public
  [HN Algolia API](https://hn.algolia.com/api): 16,490 currently indexed stories
  tagged `show_hn,story`, created during June 3–October 12 inclusive (UTC).
  Requests used `search_by_date`, `hitsPerPage=1000`, and non-overlapping
  `created_at_i>=start,created_at_i<end` filters, splitting windows when their
  hit count exceeded the returned results. IDs were deduplicated.
- Success means at least one archived rank of 30 or better during the first
  72 hours after publication. This measures an observed front-page appearance,
  not eventual points, sustained exposure, comments, or installations.
- Retain only submissions whose entire first 72 hours are inside the archive
  and do not overlap a collection gap longer than one hour. The three such gaps
  last 13.5 hours on June 25, 4.25 hours on July 25, and 145.75 hours from
  September 25 to October 1. This leaves 13,899 indexed Show HN submissions,
  of which 737 (5.30%) were observed in the top 30 within 72 hours.
- All posting windows below are Moscow time, UTC+3. Each submission counts once;
  repeated snapshots do not inflate the sample size.

## Results

| Publication time, Moscow | Submissions | Top 30 within 72h | Observed share | 95% Wilson interval |
|---|---:|---:|---:|---:|
| 00:00–06:00 | 2,196 | 131 | 5.97% | 5.05–7.04% |
| 06:00–12:00 | 1,926 | 65 | 3.37% | 2.66–4.28% |
| 12:00–18:00 | 4,853 | 230 | 4.74% | 4.18–5.37% |
| 18:00–24:00 | 4,924 | 311 | 6.32% | 5.67–7.03% |

The evening/daytime pattern persists when splitting the period:

| Period | 12:00–18:00 | 18:00–24:00 |
|---|---:|---:|
| June–July | 110/2,222 (4.95%) | 162/2,287 (7.08%) |
| August–October | 120/2,631 (4.56%) | 149/2,637 (5.65%) |

Among 4,889 submissions linking to a URL containing `github.com/`, the daytime
share is 82/1,654 (4.96%) and the evening share is 129/1,796 (7.18%). This URL
filter is a rough match for omut's intended submission, not topic classification.

Sunday has the highest whole-day share, 100/1,491 (6.71%), but its 95% interval
is 5.55–8.09%. Thursday is 128/2,310 (5.54%), interval 4.68–6.55%. The intervals
overlap. Individual weekday/hour combinations are smaller and noisier: Sunday
18:00–21:00 is 16/310 (5.16%), Thursday at those hours is 40/512 (7.81%). Do not
choose a precise slot by taking the largest of many inspected cells.

## Limits and interpretation

- This is observational. Topic, author, product quality, moderation and other
  differences between posting times are not controlled. The observed shares
  are not a forecast for omut and do not establish a causal timing effect.
- The archive alone has selection bias because it sees only the top 200.
  Joining indexed submissions supplies a denominator, but Algolia can omit
  deleted stories or posts whose titles/tags changed. Among otherwise eligible
  archived Show HNs, 40 IDs are absent from the current Show HN cohort; 33 had
  reached the top 30 within 72 hours. A checked example had lost both its title
  prefix and its Show HN tag. Results therefore describe the current indexed
  Show HN cohort, not a complete historical census of everything ever submitted
  with that prefix.
- A 15-minute sample can miss short front-page visits. Gaps of an hour or less
  remain. Posts promoted after 72 hours are outside this outcome.
- Wilson intervals treat submissions as independent and do not account for
  day-level clustering, multiple comparisons or systematic indexing omissions.
- The archive ends October 12. The US changed from daylight saving time on
  November 1; Moscow did not. Matching a US local-time window in November
  would shift the corresponding Moscow window one hour later. This further
  argues for a broad evening window rather than a supposedly optimal minute.
- Stay available after posting. A convenient evening is a better practical
  choice than chasing a noisy peak at 03:00 Moscow time.

No post was published as part of this analysis. The source database was opened
with SQLite `mode=ro`; no collector, remote machine or source-project file was
changed. Intermediate calculations and API responses are kept in this checkout's
ignored local meta, not distributed with the repository.

## Likely reaction to the project

A follow-up on 2026-11-05 used the same 16,490 indexed submissions to find a
broad thematic cohort, rather than only successful launches. A title had to
match both expressions (case-insensitive):

- `\bmemor(?:y|ies)\b|\bcontext\b|\bhandoff\b|agents\.md|claude\.md`
- `\bagents?\b|\bAI\b|\bLLMs?\b|\bClaude\b|\bCodex\b|\bMCP\b`

This returns 291 submissions: median 2 points and 0 comments; 193 have no
comments, 98 have at least one, 18 have at least five, and 13 have at least ten.
Comments include the submitter's introduction and replies, so any-comment counts
are not counts of independent reader reactions. This keyword cohort includes
adjacent use cases and repeated launches, misses differently named projects, and
is not a probability estimate for omut.

Three directly inspected examples illustrate the range:

| Submission | Points | Comments |
|---|---:|---:|
| [Recall: local project memory for Claude Code](https://news.ycombinator.com/item?id=48622590) | 138 | 85 |
| [Engrim: universal local-first memory for AI CLIs](https://news.ycombinator.com/item?id=49594008) | 93 | 63 |
| [DaiDocs: portable plain-text AI memory](https://news.ycombinator.com/item?id=49715672) | 8 | 1 (the submitter's introduction) |

The Recall discussion includes readers who already maintain short context files,
and others who prefer letting an agent inspect the code rather than maintaining
extra documentation. Engrim readers question memory quality, when records are
written, lifecycle management, evaluation, and recovery of unfinished work.
These are observed objections in those discussions, not predictions of what any
particular commenter will say about omut.

Product assessment: omut is ready to seek feedback, and adjacent launches show
that the subject can attract discussion. Its concrete reasons to try are small,
readable project records, continuity across agents and people, existing private
adoption, and deterministic shape/freshness checks. Markdown and portability by
themselves are common claims in this cohort. A worked example of a handoff is
stronger evidence of omut's usefulness than another abstract memory description.
Shape/freshness checks do not prove the truth or relevance of what an agent
records. A quiet submission would remain compatible with both a useful product
and the observed base rate of low attention.
