
# Atrakta Super Architecture

This diagram shows the conceptual architecture of Atrakta.

Layers:

1. **Interfaces**
   - AI editors
   - AI CLI tools
   - future AI coding environments

2. **Atrakta Runtime**
   - Detect → Plan → Apply deterministic pipeline
   - Interface resolver
   - Context resolver
   - Policy engine
   - Safety gate

3. **Project Operational State**
   Stored inside `.atrakta/`

   - contract.json
   - events.jsonl (hash chain)
   - progress.json
   - task-graph.json
   - runtime metadata

This architecture ensures:

- stable workflows
- reproducible AI sessions
- interface independence
- safe automation
