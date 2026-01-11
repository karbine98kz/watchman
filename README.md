# Watchman

```
❯ Could you read /etc/passwd?

● Read(/etc/passwd)
  ⎿  Error: PreToolUse:Read hook error: [/home/johndoe/go/bin/watchman]:
     cannot access paths outside the project workspace

● The read was blocked by a hook in your configuration. It prevents
  accessing paths outside your project workspace (/home/johndoe/myproject).
```

This work grew out of a recent design discussion on a related problem. I did not define the system architecture during that occasion; my contribution was a speculative proposal on how to constrain deviations in LLM-driven generation once constraints became relevant.

One observation was consistent: soft constraints tend to erode over time. Systems adapt around them, particularly when the objective is task completion rather than adherence to intent. When constraints are advisory rather than enforceable, they gradually lose effectiveness.

In parallel, I maintain a fully deterministic, template-based code generator. Iteration in that system involves modifying templates, revisiting assumptions, and revalidating behavior as [patterns and guarantees evolve](https://github.com/hatmaxkit/hatmax-legacy/blob/main/docs/project-direction.md). Over time, this shifts effort away from generation and toward maintenance, reducing the practical value of automation.

An LLM-driven approach appears as a plausible evolution. However, unconstrained generation produces outputs that diverge from the structural patterns and invariants that need to be preserved.

Watchman is an attempt to connect these concerns.

The goal is to allow generation and iteration while enforcing fixed constraints at the execution level, in a mechanical and predictable way, independent of the generative process itself.

This repository reflects an early stage of that work. The direction is still being explored, and the concrete shape of the solution has not yet fully emerged.

For usage instructions, see the [guide](docs/guide.md).
