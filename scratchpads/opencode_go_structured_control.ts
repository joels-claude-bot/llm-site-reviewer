import { createOpencodeClient } from "@opencode-ai/sdk/v2/client"
import { Agent, setGlobalDispatcher } from "undici"

const target =
  process.argv[2] ??
  "/home/claude/politician-trading-research/docs/10-decision-tree-cap-clean.md"
const directory = "/home/claude/politician-trading-research"
const serverUrl = process.env.OPENCODE_SERVER_URL ?? "http://127.0.0.1:4097"
const model = process.env.OPENCODE_MODEL ?? "opencode-go/glm-5.3-flash"
const requestTimeoutMs = Number(process.env.OPENCODE_REQUEST_TIMEOUT_MS ?? 900_000)

if (!Number.isFinite(requestTimeoutMs) || requestTimeoutMs <= 0) {
  throw new Error("OPENCODE_REQUEST_TIMEOUT_MS must be a positive number")
}

const modelSeparator = model.indexOf("/")
if (modelSeparator <= 0 || modelSeparator === model.length - 1) {
  throw new Error(`OPENCODE_MODEL must be provider/model, got ${JSON.stringify(model)}`)
}

// Node's fetch uses Undici, whose default headers timeout is shorter than some
// agent runs. OpenCode does not send response headers until the prompt finishes.
const dispatcher = new Agent({
  headersTimeout: requestTimeoutMs,
  bodyTimeout: requestTimeoutMs,
})
setGlobalDispatcher(dispatcher)

const schema = {
  type: "object",
  properties: {
    results: {
      type: "array",
      items: {
        type: "object",
        properties: {
          id: { type: "integer" },
          name: { type: "string" },
          success: { type: "boolean" },
          errors: {
            type: "array",
            items: {
              type: "object",
              properties: {
                category: { type: "string" },
                notes: { type: "string" },
              },
              required: ["category", "notes"],
              additionalProperties: false,
            },
          },
        },
        required: ["id", "name", "success", "errors"],
        additionalProperties: false,
      },
    },
  },
  required: ["results"],
  additionalProperties: false,
}

const request = {
  model: {
    providerID: model.slice(0, modelSeparator),
    modelID: model.slice(modelSeparator + 1),
  },
  format: {
    type: "json_schema" as const,
    schema,
    retryCount: 2,
  },
  parts: [
    {
      type: "text" as const,
      text: `Review ${target} for genuine reader-facing content defects.

Use other Markdown files in the repository as evidence. Find pages discussing the same entities, dates, quantities, performance figures or factual claims, and compare their full text with the target. Report clear contradictions, incorrect claims, stale references and malformed content. Do not assume only explicitly linked pages are relevant.

Base every finding only on repository content. Do not browse the web or assess rendering. Ignore stylistic preferences, harmless rounding differences and claims covering different periods or scopes.

Return exactly one result for the target: id 0, name ${JSON.stringify(target.split("/").at(-1))}, success false when errors are present and true otherwise. Put at most five concrete findings, ordered by importance, in that result's errors array. Use uppercase category codes such as CONTRADICTS_PAGE, WRONG_CLAIM, STALE_REF or OTHER.`,
    },
  ],
}

async function main() {
  console.log("REQUEST")
  console.dir(request, { depth: null })

  console.log(`SERVER ${serverUrl}`)
  console.log(`REQUEST TIMEOUT ${requestTimeoutMs}ms`)

  const client = createOpencodeClient({ baseUrl: serverUrl })
  const session = await client.session.create({
    directory,
    title: "OpenCode Go structured-output control",
  })
  if (!session.data) {
    console.error("SESSION ERROR")
    console.dir(session.error, { depth: null })
    process.exitCode = 1
    return
  }

  const response = await client.session.prompt({
    sessionID: session.data.id,
    directory,
    ...request,
  })
  if (!response.data) {
    console.error("PROMPT ERROR")
    console.dir(response.error, { depth: null })
    process.exitCode = 1
    return
  }

  console.log("RESPONSE")
  console.dir(response.data, { depth: null })
}

main()
  .catch((error: unknown) => {
    console.error("UNEXPECTED ERROR")
    console.dir(error, { depth: null })
    process.exitCode = 1
  })
  .finally(() => dispatcher.close())
