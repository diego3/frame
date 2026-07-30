# ADR-013: Technology options for a specialized sprite/animation image-generation agent

## Status

Proposed. This ADR is an options/cost survey to support a go/no-go and tool-selection decision,
not a committed design — no code in this repo depends on any of these choices yet.

## Context

The player character's spritesheet (`games/metalslug_demo/assets/vacaroxa--generic-run-n-gun-pack--v.1.0/Player/SpriteSheet_player.png`)
covers idle/run/jump/crouch/fall/death-fall poses but has no dedicated **victory** or **death**
animation. The ask: could a specialized agent generate new frames that extend this spritesheet in
its existing pixel-art style (same ~32–40px proportions, red/blue palette, black outline)?

Claude Code itself has no image-generation tool — it can read/describe the existing sheet but
cannot draw new pixel-perfect frames matching it. Standing up that capability means composing two
separate decisions:

1. **Which image-generation backend** actually produces the pixels.
2. **Which agent/orchestration layer** (if any) drives the generate → crop → validate → retry loop
   around that backend.

These are independent axes — any backend in Part 1 can in principle be driven by any orchestration
choice in Part 2 — so they're compared separately below, then recombined in the Recommendation.

**Licensing note (unresolved, see Open Questions):** no license file ships with the
`vacaroxa--generic-run-n-gun-pack--v.1.0` directory in this repo, and the other art source
(`assets/Sideview Sci-Fi - Patreon Collection.zip`) is a Patreon-distributed pack — both need their
original license terms checked before training on or materially editing their art with AI tooling.
This ADR surveys technical options only; it does not clear that licensing question.

---

## Part 1: Image-generation backend options

### Option A: OpenAI Images API (GPT Image 1.5 / GPT Image 2)

Hosted REST API. GPT Image 2 (current flagship as of mid-2026; GPT Image 1 is deprecating
2026-10-23) accepts a text prompt plus **up to 16 reference images**, at 1K/2K/4K resolution and
several aspect ratios, and supports an edit/inpaint mode. Pricing is pay-per-image, roughly
$0.005–$0.21 depending on model/quality/size.

- **Style consistency:** Reference-image conditioning can nudge output toward an existing sheet's
  palette/silhouette, but there's no fine-tuning — no guarantee of pixel-exact frame size, palette
  indices, or outline weight matching the vacaroxa pack. Best treated as a rough-draft generator a
  human then cleans up in a pixel editor (Aseprite, etc.), not a drop-in final asset source.
- **Integration effort:** Lowest of all options — one hosted HTTP call, no GPU/hosting to manage.
- **Licensing:** OpenAI's API terms grant the caller rights to output images, but the model's own
  training data provenance is not fully disclosed — a consideration if this game is ever sold
  commercially, separate from the vacaroxa/Patreon pack question above.

### Option B: Google Gemini image generation (Vertex AI / Gemini API)

Comparable hosted, multi-reference-image-conditioned generation from Google. Similar integration
shape and cost tier to Option A. Only clearly differentiated from Option A if the orchestration
layer is also Google ADK (Part 2, Option 3) — otherwise it's a close substitute for OpenAI's API
with no engine-specific advantage.

### Option C: Stable Diffusion (SDXL / Pony Diffusion XL) + a custom pixel-art LoRA

Self-hosted (ComfyUI or Automatic1111, local GPU or a rented one) or hosted inference (Replicate,
fal.ai, Civitai). The community has a mature ecosystem of pixel-art LoRAs (8-bit, 16-bit,
isometric, ...), and — more importantly for this specific ask — **a small LoRA (rank 4–8) can be
trained directly on 20–30 cropped frames from the existing `SpriteSheet_player.png`**, which is the
only option here that can target *this pack's exact style* rather than "pixel art in general."
img2img/ControlNet pose conditioning off an existing frame (e.g. the run cycle) can also anchor new
poses (a victory stance, a death/ragdoll fall) to the same proportions and camera angle.

- **Style consistency:** Highest of the three — the only option that can be fine-tuned on this
  exact asset rather than prompted toward "pixel art" generically.
- **Integration effort:** Highest — a training pipeline (crop reference frames, train LoRA, run
  img2img generation), plus the frame-stitching step every approach needs (per community guidance,
  generating a whole sprite sheet in one pass is unreliable — generate individual frames and stitch
  them with Pillow, same as this repo would do for any backend).
- **Cost:** Cheapest per-image at inference time; the LoRA training run is the main one-time cost
  (minutes to low hours on a rented GPU, or a hosted training job on Replicate/fal.ai).
- **Licensing:** Cleanest if the underlying checkpoint/LoRA licenses are checked (most SDXL
  community LoRAs are permissively licensed, but verify per-LoRA).

### Option D: Midjourney — excluded

No official API; automation is Discord-bot-only and against Midjourney's terms of service for
programmatic use. Not viable for an automated agent pipeline. Excluded from the comparison below.

### Option E: Adobe Firefly

Has a real API and commercially-safe training provenance (trained on licensed/Adobe Stock content),
which could matter if this game is ever sold and asset provenance is audited. Weaker community
tooling for pixel-art specifically compared to the SDXL LoRA ecosystem, and it's a third paid API
to integrate for a niche this repo doesn't otherwise need. Worth a mention if commercial provenance
becomes a hard requirement; not recommended as a first pick otherwise.

### Comparison at a glance

| | A: OpenAI Images | B: Google Gemini images | C: SDXL + custom LoRA | E: Adobe Firefly |
|---|---|---|---|---|
| Style match to *this* pack | Rough (reference-image only) | Rough (reference-image only) | **High** (LoRA trained on the actual sheet) | Rough |
| Setup effort | Lowest | Lowest | Highest (training pipeline) | Low |
| Per-image cost | $0.005–$0.21 | Similar tier | Lowest at inference | Similar tier |
| Needs GPU hosting | No | No | Yes (self-host or rented) | No |
| Commercial-provenance safety | Unclear | Unclear | Depends on checkpoint/LoRA | **Best** |
| Automatable via API | Yes | Yes | Yes | Yes |

---

## Part 2: Agent/orchestration SDK options

The pipeline this needs, regardless of backend, is narrow: **generate a candidate frame → crop/align
it to the sheet's existing grid → check it against the reference style (dimensions, palette,
silhouette) → either accept or regenerate → (recommended) hold for human approval before merging
into the game's assets.** That's a bounded retry loop with one human gate, not an open-ended
multi-agent collaboration problem.

1. **No framework — a plain script** (Python, using Pillow for cropping/compositing and a single
   API call to whichever Part 1 backend is chosen). Matches this repo's own standard of not adding
   abstraction beyond what a task needs (see CLAUDE.md's coding standards). Easiest to debug, no
   new dependency, easiest to hand off as a one-off tool rather than a maintained service.
2. **LangGraph** — models the retry/critique loop as an explicit graph with checkpoints and
   interrupt points, which maps directly onto "generate → critique → regenerate → human-approval
   gate" if that loop needs to be resumable or auditable. Vendor-neutral: works the same whether
   the backend is OpenAI, Google, or a self-hosted SDXL endpoint, so switching Part 1 options later
   doesn't require re-platforming the orchestration.
3. **Google ADK (Agent Development Kit)** — code-first, lower boilerplate than LangGraph, built-in
   dev UI, and a natural fit *if and only if* Option B (Gemini images) is the chosen backend and the
   project is otherwise willing to take on a GCP-shaped dependency. This repo has no existing GCP
   footprint, so that's a new platform dependency purely for this one pipeline.
4. **OpenAI Agents SDK** — same tradeoff as ADK but for Option A (OpenAI images): lightweight
   tool-calling loop, best justified if committing specifically to OpenAI as the backend.
5. **Claude Agent SDK** — Anthropic has no image-generation model, so this SDK would only be the
   "brain" that calls out to whichever Part 1 backend is chosen as a tool, plus deterministic
   Pillow post-processing as other tools. Workflow-familiar (this project is already worked on
   inside Claude Code) but functionally equivalent to LangGraph for this narrow a pipeline — no
   distinct advantage that offsets adding another SDK dependency.
6. **CrewAI / AutoGen (multi-agent frameworks)** — designed for multiple cooperating role-based
   agents (e.g. researcher + writer + reviewer). This task is a single-purpose generate/validate
   pipeline, not a multi-agent coordination problem, so these add conceptual and operational
   overhead — separate agent roles, inter-agent messaging — with no corresponding benefit here.
   Considered, not recommended.

### Comparison at a glance

| | Plain script | LangGraph | Google ADK | OpenAI Agents SDK | Claude Agent SDK | CrewAI/AutoGen |
|---|---|---|---|---|---|---|
| Fit for a single bounded retry loop | **Best** | Good | Good (Gemini-locked) | Good (OpenAI-locked) | Good | Overkill |
| Backend-agnostic | Yes | **Yes** | No (Gemini/GCP) | No (OpenAI) | Yes (tool-call only) | Yes |
| New dependency/platform footprint | None | One library | GCP + ADK | OpenAI SDK | Anthropic SDK | Two libraries + coordination overhead |
| Human-in-the-loop / resumable checkpoints | Manual | **Built-in** | Partial | Partial | Manual | Partial |

---

## Recommendation

**Backend:** Start with **Option C (SDXL + a small custom LoRA trained on the existing
`SpriteSheet_player.png` frames)** for anything meant to ship in-game, since style-consistency to
this exact pack — not "pixel art in general" — is the actual requirement, and only a fine-tuned
LoRA can target that. Use **Option A (OpenAI Images)** as a same-day, low-effort path to a rough
draft for validating frame *count* and *timing* (how many frames a victory/death animation needs)
before investing in LoRA training and img2img pose conditioning.

**Orchestration:** Start with **no framework** — a plain Python pipeline script (generate → crop to
the sheet's existing frame grid → human visual approval before merging). This is a narrow,
single-purpose tool, and matches this repo's existing anti-over-engineering standard (CLAUDE.md:
"don't add abstraction beyond what the task requires"). Only promote to **LangGraph** if the
pipeline grows real branching state — e.g. an automatic critique-and-regenerate loop using a
vision-capable model to score style match, multiple asset categories (enemies, weapons, UI) sharing
the same pipeline, or a need for resumable/auditable runs.

**Explicitly not recommended:** Midjourney (no automatable API), and CrewAI/AutoGen or a
GCP-/OpenAI-locked agent SDK (ADK, OpenAI Agents SDK) as a *first* choice — each solves a
coordination or platform-lock-in problem this task doesn't have yet.

## Open questions before starting

- **Licensing:** confirm the vacaroxa pack's and the Patreon sci-fi pack's license terms permit
  AI-assisted derivative frames and (for Option C) training a LoRA on their art. No license file
  ships in either asset directory in this repo today — this needs to be checked against the
  original marketplace/Patreon listing before any training run.
- GPU hosting choice for Option C if pursued (existing local GPU vs. a rented Replicate/fal.ai
  training + inference job) — affects cost and turnaround time.
- Full scope of new animations wanted beyond victory/death (hurt/hit-reaction, taunt, etc.) —
  affects whether LoRA training investment pays off vs. one-off hosted-API generations.
- Where the human-approval gate lives concretely — e.g. new frames land in a `assets/_review/`
  folder and only get wired into `SpriteSheet_player.png`/the `Animator` component after manual
  approval, keeping this in line with the **No Regression Rule** (golden-path testing before any
  new asset ships).

## References

- `games/metalslug_demo/assets/vacaroxa--generic-run-n-gun-pack--v.1.0/Player/SpriteSheet_player.png` — the asset in question.
- ADR-012: same "options/cost survey, not a committed design" format this ADR follows.
- CLAUDE.md coding standards — "don't add abstraction beyond what the task requires," cited above
  for the orchestration recommendation.
