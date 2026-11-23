import type { Register } from "claude-code";

// The agent panel's rows on Claude Code 2.1.326+, where a mod replaces the
// deprecated subagentStatusLine setting. Rendering stays in Go: each row is
// sent to `howl --subagent` in that setting's schema, so there is one renderer
// and an older installed binary already understands the request.

// Long enough for a first launch; on timeout the row keeps Claude Code's look.
const TIMEOUT_MS = 2000;

type Drawn = { input: string; body: Promise<string | undefined> };

function bodyFor(id: string, stdout: string): string | undefined {
  for (const line of stdout.split("\n")) {
    if (!line) continue;
    const row: { id: string; content: string } = JSON.parse(line);
    if (row.id === id) return row.content;
  }
  return undefined;
}

export const register: Register = (on) => {
  // Every row is asked again when any row comes or goes, finished ones
  // included. Only a change in what Howl reads is worth another process.
  const drawn = new Map<string, Drawn>();

  on("ui.render", { component: "AgentRow" }, async ($, e, next) => {
    const { props } = e;
    // A body that is already set comes from a subagentStatusLine setting or
    // another mod. Other surfaces keep their own rows, as with that setting.
    if (e.surface !== "terminal" || props.isHidden) return next(e);
    if (props.body !== undefined) return next(e);

    const home = await $.env.get("HOME");
    if (!home) return next(e);

    const input = JSON.stringify({
      columns: props.columns,
      tasks: [
        {
          id: props.id,
          name: props.name,
          type: props.kind,
          // The setting carried the task's status, which is not props.status.
          status: props.taskStatus,
          description: props.description,
          label: props.label,
          model: props.model,
          effort: props.effort,
          contextWindowSize: props.contextWindowSize,
          tokenCount: props.tokenCount
        }
      ]
    });

    let last = drawn.get(props.id);
    if (last?.input !== input) {
      const run = $.process.run(
        [`${home}/.claude/hud/howl`, "--subagent"],
        // Not the session's directory: that can be a worktree since removed.
        { stdin: input, cwd: home, timeoutMs: TIMEOUT_MS }
      );
      const body = run.then(({ exitCode, stdout }) => {
        // A failed exit resolves too, as when the binary is being replaced.
        if (exitCode !== 0) throw new Error(`howl exited ${exitCode}`);
        return bodyFor(props.id, stdout);
      });
      // Stored before it settles, so overlapping asks share one process.
      last = { input, body };
      drawn.set(props.id, last);
    }
    const body = await last.body.catch((err: unknown) => {
      // A failure is retried on the next ask, unless a newer one replaced it.
      if (drawn.get(props.id) === last) drawn.delete(props.id);
      throw err;
    });

    // An empty body would hide the row, so no content means the default row.
    if (!body) return next(e);
    return next({ ...e, props: { ...props, body } });
  }).catch(($, e, next) => next(e)); // no binary, a timeout, or unreadable output
};
