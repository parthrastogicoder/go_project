
---

# P2P Encrypted Chat

This project is a peer-to-peer (P2P) chat application that supports **end-to-end encrypted messaging** using **WebRTC** for direct peer connections and **AES-GCM encryption** for message security.

At a high level, the system consists of:
- A **signaling server** written in Go, which uses WebSockets to facilitate the initial exchange of WebRTC offers and answers between peers.
- A **P2P chat client** (in both Go and browser-based JavaScript) that establishes a secure, direct connection using WebRTC and transmits encrypted messages.
- A **browser UI** for chatting, allowing users to choose their role (offerer/answerer), type messages, and exchange them securely with another peer.

All messages sent between peers are encrypted using a shared AES key, ensuring confidentiality and security over the network.

This application serves as a minimal working example of building an encrypted chat tool using open web standards and low-level cryptographic primitives, suitable for learning or prototyping secure communications.

---
