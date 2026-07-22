# StagePoster Backend Architecture

## Overview

StagePoster uses three independently managed services:

```text
Go Backend
ComfyUI
vLLM
They cooperate through local HTTP APIs.
System Diagram
┌──────────────────────────────┐
│ Frontend                     │
│ Web / Desktop / Mobile       │
└──────────────┬───────────────┘
               │ HTTPS
               ▼
┌──────────────────────────────┐
│ Cloudflare Tunnel            │
│ Development public gateway   │
└──────────────┬───────────────┘
               │ localhost:8080
               ▼
┌──────────────────────────────┐
│ Go Backend                   │
│ Session and poster runtime   │
├──────────────────────────────┤
│ Conversation state machine   │
│ Design plan persistence      │
│ Candidate orchestration      │
│ Deterministic composer       │
│ Review loop                  │
│ Snapshot and restore         │
│ SQLite repository            │
└───────────┬───────────┬──────┘
            │           │
            │           │
            ▼           ▼
┌─────────────────┐   ┌─────────────────┐
│ ComfyUI :8188   │   │ vLLM :8001      │
│ Z-Image-Turbo   │   │ Qwen3.5-9B      │
│ Image generation│   │ Plan and review │
└─────────────────┘   └─────────────────┘
Responsibilities
Go Backend
The Go service owns:
AI session state
Conversation history
Missing requirement fields
Design plan persistence
Candidate generation requests
Candidate selection
Poster composition
Review records
Review snapshots
Best-so-far restoration
Finalization idempotency
Public API and CORS
Database and storage paths
ComfyUI
ComfyUI owns:
Z-Image-Turbo inference
Prompt execution
Seed execution
Job status
Generated key visual files
ComfyUI does not own:
User sessions
Text layout
Event metadata
Review history
Finalization
vLLM
vLLM serves Qwen3.5-9B through an OpenAI-compatible API.
The model is used for:
Requirement extraction
Follow-up questions
Design-plan generation
Multimodal poster review
Structured review decisions
The current runtime uses vLLM sleep mode to release GPU memory when the VLM is idle.
GPU Lifecycle
The Radeon GPU is shared by ComfyUI and vLLM.
Conversation or review request
        ↓
Wake VLM
        ↓
Run Qwen3.5-9B
        ↓
Sleep VLM and release VRAM
        ↓
ComfyUI generation may run
The vLLM sleep endpoints are bound to localhost and must not be exposed publicly.
AI Session State
Frontend code should primarily follow availableActions.
Typical state progression:
collecting requirements
        ↓
awaiting plan confirmation
        ↓
generating candidates
        ↓
awaiting candidate selection
        ↓
succeeded
        ↓
finalize
        ↓
succeeded or completed_with_warnings
Additional terminal or exceptional states include:
needs_user_input
failed
cancelled
Poster State
Typical poster progression:
created
generating_candidates
awaiting_selection
selected
composing
succeeded
failed
Review Loop
Final poster V1
        ↓
Review Round 1
        ↓
Decision
   ├── ACCEPT
   ├── RECOMPOSE
   ├── REGENERATE
   └── REWRITE_BRIEF
        ↓
Snapshot current round
        ↓
Apply action
        ↓
Review next version
        ↓
Maximum rounds or ACCEPT
        ↓
Restore highest-scoring snapshot
RECOMPOSE vs REGENERATE
RECOMPOSE
Reuses the selected candidate key visual
Adjusts deterministic text and layout
Does not run ComfyUI again
Uses CompositionAdjustments
REGENERATE
Extends positive and negative prompts
Submits a new ComfyUI generation
Adopts the regenerated job into the selected candidate
Composes the new key visual
Runs another review
Composition Adjustments
The current deterministic composer supports:
Template
TitleOffsetRatio
PanelTopRatio
PanelTheme
Review issue codes are mapped to these parameters.
Examples:
TITLE_COLLISION
INFORMATION_PANEL_CONTRAST
INFORMATION_PANEL_POSITION
SPACING
HIERARCHY
Persistence
SQLite tables include:
ai_sessions
ai_messages
ai_design_plans
ai_session_assets
assets
poster_requests
poster_candidates
poster_outputs
poster_reviews
jobs
outputs
Review images are preserved as filesystem snapshots:
review-round-1-final_poster.png
review-round-1-thumbnail.png
review-round-2-final_poster.png
review-round-2-thumbnail.png
Idempotency
Finalize is idempotent.
After a session reaches a finalized state:
A repeated finalize call returns the existing result
No additional review round is created
The final image is not changed
No duplicate composition is performed
Concurrency
Finalize calls are serialized by session key.
Calls for the same session cannot modify the poster concurrently.
Different sessions may finalize independently.
Public Exposure
Only this service should be exposed:
Go Backend :8080
These must remain private:
ComfyUI :8188
vLLM :8001
SQLite files
Storage directories
