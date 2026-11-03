import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { setTimeout as delay } from "node:timers/promises";

const origin = process.env.SMOKE_ORIGIN ?? "http://127.0.0.1:8080";
const apiKey = process.env.SMOKE_API_KEY ?? "local-key";
function compose(...args) {
  const result = spawnSync("docker", ["compose", ...args], {
    encoding: "utf8",
    timeout: 120000,
  });
  assert.equal(
    result.status,
    0,
    result.stderr || result.stdout || result.error?.message,
  );
  return result.stdout.trim();
}
async function waitFor(check, description, seconds = 90) {
  for (let n = 0; n < seconds; n++) {
    try {
      if (await check()) return;
    } catch {
      /* The service may still be starting. */
    }
    await delay(1000);
  }
  throw new Error("Timed out waiting for " + description);
}
const ready = () =>
  waitFor(async () => (await fetch(origin + "/readyz")).ok, "API readiness");
async function api(path, { status = 200, method = "GET", body, key } = {}) {
  const response = await fetch(origin + path, {
    method,
    headers: {
      "X-API-Key": apiKey,
      "Content-Type": "application/json",
      ...(key ? { "Idempotency-Key": key } : {}),
    },
    ...(body ? { body: JSON.stringify(body) } : {}),
  });
  const data = await response.json();
  assert.equal(response.status, status, JSON.stringify(data));
  return { data, response };
}
function sql(query) {
  return compose(
    "exec",
    "-T",
    "postgres",
    "psql",
    "-U",
    "stockledger",
    "-d",
    "stockledger",
    "-Atc",
    query,
  );
}

await ready();
assert.equal((await fetch(origin + "/v1/items")).status, 401);
const apiContainer = compose("ps", "-q", "api");
const inspected = spawnSync(
  "docker",
  ["inspect", "--format", "{{.Config.User}}", apiContainer],
  { encoding: "utf8" },
);
assert.equal(inspected.status, 0);
assert.equal(inspected.stdout.trim(), "nonroot:nonroot");
await api("/v1/items", {
  method: "POST",
  status: 201,
  body: { sku: "docker-smoke-desk", available: 5 },
});
const item = (await api("/v1/items")).data.find(
  (row) => row.sku === "docker-smoke-desk",
);
assert.ok(item?.id);
const command = { item_id: item.id, quantity: 2 };
const first = await api("/v1/reservations", {
  method: "POST",
  status: 201,
  key: "docker-checkout-1",
  body: command,
});
const replay = await api("/v1/reservations", {
  method: "POST",
  status: 201,
  key: "docker-checkout-1",
  body: command,
});
assert.equal(replay.data.id, first.data.id);
assert.equal(replay.response.headers.get("Idempotent-Replay"), "true");
await api("/v1/reservations", {
  method: "POST",
  status: 409,
  key: "docker-checkout-1",
  body: { ...command, quantity: 3 },
});
assert.equal(
  (await api("/v1/items")).data.find((row) => row.id === item.id).available,
  3,
);
await api("/v1/reservations/" + first.data.id + "/release", { method: "POST" });
await api("/v1/reservations/" + first.data.id + "/release", {
  method: "POST",
  status: 409,
});
assert.equal(
  (await api("/v1/items")).data.find((row) => row.id === item.id).available,
  5,
);
const second = await api("/v1/reservations", {
  method: "POST",
  status: 201,
  key: "docker-checkout-2",
  body: { ...command, quantity: 3 },
});
await api("/v1/reservations/" + second.data.id + "/confirm", {
  method: "POST",
});
await api("/v1/reservations/" + second.data.id + "/release", {
  method: "POST",
  status: 409,
});
assert.equal(
  (await api("/v1/items")).data.find((row) => row.id === item.id).available,
  2,
);
await waitFor(
  () =>
    sql("SELECT count(*) FROM outbox_events WHERE published_at IS NOT NULL") ===
    "4",
  "all four outbox events to reach Kafka",
);
const messages = compose(
  "exec",
  "-T",
  "kafka",
  "/opt/kafka/bin/kafka-console-consumer.sh",
  "--bootstrap-server",
  "kafka:29092",
  "--topic",
  "stockledger.events",
  "--from-beginning",
  "--max-messages",
  "4",
  "--timeout-ms",
  "30000",
  "--property",
  "print.key=true",
);
const lines = messages.split("\n").filter(Boolean);
assert.equal(lines.length, 4);
const seen = new Set();
for (const line of lines) {
  const separator = line.indexOf("\t");
  assert.ok(separator > 0, "Kafka event must have a stable message key");
  const eventId = line.slice(0, separator);
  const payload = JSON.parse(line.slice(separator + 1));
  assert.match(eventId, /^[0-9a-f-]{36}$/);
  assert.ok([first.data.id, second.data.id].includes(payload.reservation_id));
  seen.add(eventId);
}
assert.equal(seen.size, 4);
assert.match(
  await (await fetch(origin + "/metrics")).text(),
  /stockledger_http_requests_total/,
);
compose("restart", "api");
await ready();
assert.equal(
  (await api("/v1/items")).data.find((row) => row.id === item.id).available,
  2,
);
compose("down"); // The dedicated CI project's named database volume is preserved.
compose("up", "-d", "--wait", "--wait-timeout", "120");
await ready();
assert.doesNotMatch(compose("logs", "--no-color", "migrate"), /ERROR:/);
assert.equal(
  (await api("/v1/items")).data.find((row) => row.id === item.id).available,
  2,
);
assert.equal(sql("SELECT count(*) FROM reservations"), "2");
console.log(
  "Docker smoke passed: nonroot API, auth, stock/replay/conflict, release/confirm, four Kafka events, metrics, restart and volume persistence.",
);
