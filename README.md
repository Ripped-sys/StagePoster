# Poster Engine

AI Multimodal Poster Generation System.

## Architecture

User
 |
Poster API
 |
LLM Planner
 |
ComfyUI Runtime
 |
Z-Image Turbo
 |
Generated Poster


## Components

- LLM planning
- Image generation
- Workflow orchestration
- Review loop
- Asset management

## Current Status

- AMD W7900 ROCm inference ready
- Z-Image Turbo deployed
- ComfyUI workflow ready


## MVP-0: Poster Generation Pipeline

Goal:
输入:
- artist
- event info
- logo
- style

输出:
- generated poster

Pipeline:
API
 ↓
Prompt Builder
 ↓
ComfyUI
 ↓
Image Generation
 ↓
Result Storage
