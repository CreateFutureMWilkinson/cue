# UI Navigation Map

Diagram of all views and navigation flows in the Cue UI.

```mermaid
flowchart TD
    %% ── Styles ──
    classDef character fill:#eebefa,stroke:#8b5cf6,stroke-width:2px,color:#1e1e1e
    classDef notif fill:#ffc9c9,stroke:#ef4444,stroke-width:2px,color:#1e1e1e
    classDef plan fill:#d3f9d8,stroke:#22c55e,stroke-width:2px,color:#1e1e1e
    classDef noplan fill:#fff3bf,stroke:#f59e0b,stroke-width:2px,color:#1e1e1e
    classDef wizard fill:#a5d8ff,stroke:#4a9eed,stroke-width:2px,color:#1e1e1e
    classDef modal fill:#ffd8a8,stroke:#f59e0b,stroke-width:2px,stroke-dasharray:5 5,color:#1e1e1e

    %% ── Full-screen views ──
    Main["<b>Main / Character View</b><br/>Focus Rail 10% | Character 60% | Notifs 30%<br/>Activity Log drawer · Default startup"]:::character

    Expanded["<b>Expanded Notifications</b><br/>Focus Rail 10% | Notifications 90%<br/>Full-width cards + Dismiss<br/>Review button in rail · Character hidden"]:::notif

    PlanActive["<b>Plan View (Active Plan)</b><br/>Schedule tree + Todo List (50/50)<br/>Abandon Plan button<br/>Elapsed blocks pruned"]:::plan

    PlanNone["<b>Plan View (No Plan)</b><br/>Snarky placeholder + Todo List (50/50)<br/>Plan My Day button"]:::noplan

    Wiz1["<b>Step 1: Select Tasks</b><br/>Checkboxes + Add task<br/>Cancel · Next"]:::wizard
    Wiz2["<b>Step 2: Time Estimates</b><br/>Override inputs<br/>Back · Next"]:::wizard
    Wiz3["<b>Step 3: Priority Order</b><br/>Up/Down arrows<br/>Back · Next"]:::wizard
    Wiz4["<b>Step 4: Choose Schedule</b><br/>A vs B cards<br/>Back · Select"]:::wizard

    %% ── Modal dialogs ──
    NotifDetail["<b>Notification Detail</b><br/>MODAL<br/>IS · CS · Timestamp<br/>Full content · Resolve"]:::modal
    Feedback["<b>Feedback Review</b><br/>MODAL 600×400<br/>Rate 0–10 · Notes<br/>Skip · Delete"]:::modal
    TaskDetail["<b>Task Detail</b><br/>MODAL 500×450<br/>Edit: title, priority,<br/>category, due, notes"]:::modal
    Settings["<b>Settings</b><br/>MODAL 400×280<br/>Notif + Timer volume"]:::modal

    %% ── Navigation: Main ↔ Expanded Notifications ──
    Main -- "expand toggle" --> Expanded
    Expanded -- "collapse toggle" --> Main

    %% ── Navigation: Main ↔ Plan View ──
    Main -- "Plan button" --> PlanActive
    Main -- "Plan button<br/>(no plan exists)" --> PlanNone
    PlanActive -- "Back button" --> Main
    PlanNone -- "Back button" --> Main

    %% ── Navigation: Plan states ──
    PlanActive -- "Abandon Plan" --> PlanNone

    %% ── Navigation: Plan → Wizard → Plan ──
    PlanNone -- "Plan My Day" --> Wiz1
    Wiz1 -- "Next" --> Wiz2
    Wiz2 -- "Next" --> Wiz3
    Wiz3 -- "Next" --> Wiz4
    Wiz4 -- "Select A/B" --> PlanActive

    %% ── Navigation: Wizard back/cancel ──
    Wiz2 -. "Back" .-> Wiz1
    Wiz3 -. "Back" .-> Wiz2
    Wiz4 -. "Back" .-> Wiz3
    Wiz1 -. "Cancel" .-> PlanNone
    Wiz2 -. "Cancel" .-> PlanNone
    Wiz3 -. "Cancel" .-> PlanNone
    Wiz4 -. "Cancel" .-> PlanNone

    %% ── Modal triggers ──
    Main -. "card click" .-> NotifDetail
    Expanded -. "card click" .-> NotifDetail
    Expanded -. "Review button" .-> Feedback
    PlanActive -. "[details]" .-> TaskDetail
    PlanNone -. "[details]" .-> TaskDetail
    Main -. "Menu → Settings" .-> Settings
```

## Legend

| Style | Meaning |
|---|---|
| Solid border | Full-screen view (occupies center area or replaces columns) |
| Dashed border | Modal dialog (blocks main window) |
| Solid arrow | Primary navigation (forward flow) |
| Dotted arrow | Back / Cancel / modal open |
| Purple | Character view (default) |
| Green | Plan view (active plan) |
| Yellow | Plan view (no plan) |
| Blue | Wizard steps |
| Red | Expanded notifications |
| Orange | Modal dialogs |
