# Atrakta Architecture Diagram (Mermaid)

Paste this into your README (GitHub supports Mermaid).

```mermaid
graph TD

User[Developer]

subgraph Interfaces
Cursor[Cursor]
VSCode[VS Code]
ClaudeCLI[Claude Code / claude_code]
CodexCLI[Codex CLI / codex_cli]
Aider[aider]
end

User --> Cursor
User --> VSCode
User --> ClaudeCLI
User --> CodexCLI
User --> Aider

Cursor --> Harness
VSCode --> Harness
ClaudeCLI --> Harness
CodexCLI --> Harness
Aider --> Harness

subgraph Atrakta Runtime
Harness[Atrakta Runtime]

Detect[Detect]
Plan[Plan (Task DAG)]
Apply[Apply (Topo order)]
Gate[Gate (Safety + Quality)]
end

Harness --> Detect
Detect --> Plan
Plan --> Apply
Apply --> Gate

Gate --> State
Gate --> Events
Gate --> Progress
Gate --> TaskGraph
Gate --> Metrics

subgraph Project State (.atrakta/)
State[state.json]
Events[events.jsonl (append-only hash chain)]
Progress[progress.json]
TaskGraph[task-graph.json]
Metrics[metrics/runtime.json]
end
```
