# ADR-004: Networking approaches

## Status

Proposed.

## Context

We may want multiplayer: two or more players in the same game session, each on their own machine. The game state must stay in sync across machines. There are different ways to do this; each has different tradeoffs. This ADR describes two common approaches so the team can choose one when implementing networking.

**Prerequisite:** With the event-based architecture (ADR-003), the network layer fits as a **view**: it turns received messages into **intent events** (for remote players’ actions) and sends **state events** (or state snapshots) to the network. Game Logic does not know whether an intent came from the local player or from the network.

---

## Approach 1: Authoritative server

**Idea:** One machine is the **server**. It runs the real game logic and is the only source of truth. **Clients** send the server the player’s intents (e.g. “jump”, “move left”). The server runs the simulation and sends back the current **game state** (or changes to it). Clients only display what the server tells them; they do not run the rules themselves.

```
Client A (you)     →  intents  →  Server  →  state  →  Client A, B, C...
Client B (friend)  →  intents  →    ↑
Client C           →  intents  →
```

**Good for:** Competitive games, when you care about fairness and anti-cheat. Works well with our event model: clients emit intents over the wire; server runs Logic and broadcasts state events.

| Pros | Cons |
|------|------|
| Fair: everyone sees the same truth. | You need a machine running as server (dedicated or “listen server”). |
| Cheating is harder (server validates everything). | Input can feel delayed: you press jump, server decides, then you see it (latency). |
| Single place to fix bugs and add features. | More code: server build, client build, connection flow. |

**Summary:** Best when you need a single, trusted referee. You pay with server cost and latency.

---

## Approach 2: Peer-to-peer (shared simulation)

**Idea:** There is **no dedicated server**. Every player’s game runs the **same logic** and they **exchange data** so everyone stays in sync. Usually either: (a) everyone sends **inputs/intents** and each peer runs the same simulation with the same inputs (“input sync”), or (b) one peer is temporary “host” and broadcasts **state** to others who adopt it (“state sync”).

```
Peer A  ←→  intents or state  ←→  Peer B
   ↑                                  ↑
   └── same game logic on both ───────┘
```

**Good for:** Small groups (e.g. 2–4 players), co-op, or when you don’t want to run a server. Fits our model: each peer has Logic; the “network view” sends intents to remote peers and receives their intents (or state) and feeds them into the local event bus.

| Pros | Cons |
|------|------|
| No dedicated server to run or pay for. | One peer can cheat (they control their own logic). |
| Simpler deployment: everyone runs the same build. | Sync is tricky: everyone must apply inputs in the same order, or agree on whose state wins. |
| Can feel responsive (everyone simulates locally). | If one peer is slow or drops, the whole game can stall or desync. |

**Summary:** Best when you want simplicity and small player counts and can accept less control over fairness. You pay with sync complexity and vulnerability to cheating.

---

## Comparison at a glance

| | Authoritative server | Peer-to-peer |
|--|----------------------|---------------|
| **Who runs the rules?** | Only the server. | Every peer. |
| **Who is the source of truth?** | Server. | Agreement between peers (or host). |
| **Need a server?** | Yes (dedicated or listen). | No. |
| **Fairness / anti-cheat** | Strong. | Weak. |
| **Implementation complexity** | Medium (server + client + sync). | Medium (sync and ordering are hard). |
| **Latency / responsiveness** | Can feel laggy unless you add prediction (extra work). | Can feel good if sync is done well. |

---

## Recommendation

- Prefer **authoritative server** when the game is competitive or you care about cheating and a single source of truth. Plan for a small amount of latency; add client-side prediction later if needed.
- Prefer **peer-to-peer** when the game is co-op or casual, player count is low, and you want to avoid running a server. Invest in clear input ordering and state reconciliation to avoid desyncs.

Both approaches work with ADR-003: the network is a view that turns messages into intents and state events, so Game Logic stays independent of the network.

## References

- ADR-002: Event manager (Bus, Emit/Subscribe).
- ADR-003: Layer separation; Remote Game Logic / network as a view; intent vs state events.
