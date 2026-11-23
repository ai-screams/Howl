import type { On, RenderPropsOf } from "claude-code";
import { expect, mock, test } from "claude-code/testing";

// Run with `make plugin-test`. Each test names the change that fails it.

const PROPS: RenderPropsOf["AgentRow"] = {
  id: "a1",
  name: "scout",
  kind: "local_agent",
  type: "Explore",
  description: "map the repo",
  label: "reading render.go",
  status: "waiting",
  taskStatus: "completed",
  startTime: 1,
  cwd: "/repo",
  model: "some-model",
  effort: "high",
  contextWindowSize: 200000,
  tokenCount: 50000,
  tokenSamples: [50000],
  hasRow: true,
  rowIds: ["a1"],
  columns: 70,
  isHidden: false
};

type Call = {
  argv: readonly string[];
  init?: { stdin?: string; cwd?: string; timeoutMs?: number };
};

// Stands for the engine: records each process and the body that reaches it.
function engine(
  on: On,
  stdout: string | Error,
  env = { HOME: "/home/u" } as Record<string, string>
) {
  const seen: { calls: Call[]; body?: string | undefined } = { calls: [] };
  mock.env(on, env);
  on("process.run", (_, e) => {
    seen.calls.push(e);
    if (stdout instanceof Error) throw stdout;
    return {
      value: {
        exitCode: 0,
        stdout,
        stderr: "",
        isStdoutTruncated: false,
        isStderrTruncated: false
      }
    };
  });
  on("ui.render", { component: "AgentRow" }, (_, e, next) => {
    seen.body = e.props.body;
    return next(e);
  });
  return seen;
}

const row = (content: string, id = "a1") =>
  JSON.stringify({ id, content }) + "\n";

const target = (props: Partial<RenderPropsOf["AgentRow"]> = {}) =>
  ({
    plugin: "howl",
    surface: "terminal",
    component: "AgentRow",
    requestId: "a1",
    props: { ...PROPS, ...props }
  }) as const;

// Fails if: the argv, cwd, timeout, or any forwarded field changes, such as
// props.status sent in place of props.taskStatus.
test("sends the subagentStatusLine schema to the binary", async ($, on) => {
  const seen = engine(on, row("drawn"));
  await $.ui.mount(target());
  expect(seen.calls.length).toBe(1);
  const { argv, init } = seen.calls[0]!;
  expect(argv).toEqual(["/home/u/.claude/hud/howl", "--subagent"]);
  expect(init?.cwd).toBe("/home/u");
  expect(init?.timeoutMs).toBe(2000);
  expect(JSON.parse(init?.stdin ?? "")).toEqual({
    columns: 70,
    tasks: [
      {
        id: "a1",
        name: "scout",
        type: "local_agent",
        status: "completed",
        description: "map the repo",
        label: "reading render.go",
        model: "some-model",
        effort: "high",
        contextWindowSize: 200000,
        tokenCount: 50000
      }
    ]
  });
});

// Fails if: the line is not matched by id, or its content is not the body.
test("draws the content printed for this id", async ($, on) => {
  const seen = engine(
    on,
    row("someone else", "b2") + row("\u001b[36mscout\u001b[0m")
  );
  await $.ui.mount(target());
  expect(seen.body).toBe("\u001b[36mscout\u001b[0m");
});

// Fails if: the id comparison is dropped.
test("ignores content for another id", async ($, on) => {
  const seen = engine(on, row("someone else", "b2"));
  await $.ui.mount(target());
  expect(seen.body).toBe(undefined);
});

// Fails if: an empty content is passed on, which would hide the row.
test("never hides a row with an empty body", async ($, on) => {
  const seen = engine(on, row(""));
  await $.ui.mount(target());
  expect(seen.body).toBe(undefined);
});

// Fails if: blank lines reach JSON.parse.
test("skips blank lines", async ($, on) => {
  const seen = engine(on, "\n" + row("drawn"));
  await $.ui.mount(target());
  expect(seen.body).toBe("drawn");
});

// Fails if: the body guard is dropped.
test("leaves a row that something else drew", async ($, on) => {
  const seen = engine(on, row("drawn"));
  await $.ui.mount(target({ body: "theirs" }));
  expect(seen.calls.length).toBe(0);
  expect(seen.body).toBe("theirs");
});

// Fails if: the isHidden guard is dropped.
test("starts nothing for a hidden row", async ($, on) => {
  const seen = engine(on, row("drawn"));
  await $.ui.mount(target({ isHidden: true }));
  expect(seen.calls.length).toBe(0);
});

// Fails if: the surface guard is dropped.
test("leaves the other surfaces alone", async ($, on) => {
  const seen = engine(on, row("drawn"));
  for (const surface of ["desktop", "vscode", "mobile"] as const) {
    await $.ui.mount({ ...target(), surface });
  }
  expect(seen.calls.length).toBe(0);
});

// Fails if: the HOME guard is dropped.
test("starts nothing without HOME", async ($, on) => {
  const seen = engine(on, row("drawn"), {});
  await $.ui.mount(target());
  expect(seen.calls.length).toBe(0);
});

// Fails if: the cache is dropped, or keyed on more than Howl reads.
test("reuses a row while its input is unchanged", async ($, on) => {
  const seen = engine(on, row("drawn"));
  await (await $.ui.mount(target())).unmount();
  seen.body = undefined;
  await $.ui.mount(target({ rowIds: ["a1", "b2"], tokenSamples: [1, 50000] }));
  expect(seen.calls.length).toBe(1);
  expect(seen.body).toBe("drawn");
});

// Fails if: the cache is served whatever the input.
test("renders again when the tokens change", async ($, on) => {
  const seen = engine(on, row("drawn"));
  await (await $.ui.mount(target())).unmount();
  await $.ui.mount(target({ tokenCount: 60000 }));
  expect(seen.calls.length).toBe(2);
});

// Fails if: one cache slot serves every agent.
test("keeps agents apart", async ($, on) => {
  const seen = engine(on, row("first") + row("second", "b2"));
  const b = { ...target({ id: "b2" }), requestId: "b2" };
  for (const round of [1, 2]) {
    await (await $.ui.mount(target())).unmount();
    expect(seen.body).toBe("first");
    await (await $.ui.mount(b)).unmount();
    expect(seen.body).toBe("second");
    // The second round is served from the cache.
    expect(seen.calls.length, `round ${round}`).toBe(2);
  }
});

// Fails if: the hook stops passing a failed process through to the engine.
test("keeps the default row when the binary cannot run", async ($, on) => {
  const seen = engine(on, new Error("ENOENT"));
  await $.ui.mount(target());
  expect(seen.body).toBe(undefined);
});

// Fails if: unreadable output becomes a body.
test("keeps the default row when the output is not JSON", async ($, on) => {
  const seen = engine(on, "howl: stdin parse error\n");
  await $.ui.mount(target());
  expect(seen.body).toBe(undefined);
});

// Fails if: a nonzero exit is read as output, or stays cached.
test("does not draw or cache a failed exit", async ($, on) => {
  let calls = 0;
  let body: string | undefined;
  mock.env(on, { HOME: "/home/u" });
  on("process.run", () => ({
    value: {
      exitCode: ++calls === 1 ? 1 : 0,
      stdout: row("drawn"),
      stderr: "",
      isStdoutTruncated: false,
      isStderrTruncated: false
    }
  }));
  on("ui.render", { component: "AgentRow" }, (_, e, next) => {
    body = e.props.body;
    return next(e);
  });
  await (await $.ui.mount(target())).unmount();
  expect(body).toBe(undefined);
  await $.ui.mount(target());
  expect(calls).toBe(2);
  expect(body).toBe("drawn");
});

// Fails if: the cache is written only once the process has finished.
test("shares one process between overlapping asks", async ($, on) => {
  const seen = engine(on, row("drawn"));
  await Promise.all(
    [1, 2, 3, 4].map((n) => $.ui.mount({ ...target(), requestId: `a1-${n}` }))
  );
  expect(seen.calls.length).toBe(1);
});

// Fails if: a failure evicts the entry of the newer ask that replaced it.
test("a late failure leaves the newer entry cached", async ($, on) => {
  let calls = 0;
  let started = () => {};
  let finish = () => {};
  const firstStarted = new Promise<void>((resolve) => (started = resolve));
  const firstMayFail = new Promise<void>((resolve) => (finish = resolve));
  mock.env(on, { HOME: "/home/u" });
  on("process.run", async () => {
    const call = ++calls;
    if (call === 1) {
      started();
      await firstMayFail;
    }
    return {
      value: {
        exitCode: call === 1 ? 1 : 0,
        stdout: row("drawn"),
        stderr: "",
        isStdoutTruncated: false,
        isStderrTruncated: false
      }
    };
  });
  on("ui.render", { component: "AgentRow" }, (_, e, next) => next(e));

  const older = $.ui.mount(target());
  await firstStarted;
  const newer = target({ tokenCount: 60000 });
  await $.ui.mount({ ...newer, requestId: "a1-newer" });
  finish();
  await older;
  await $.ui.mount({ ...newer, requestId: "a1-again" });
  expect(calls).toBe(2);
});
