import { execFile } from "node:child_process";
import { createHash, randomBytes } from "node:crypto";
import { createServer } from "node:http";
import { promisify } from "node:util";
import { z } from "zod";

const execFileAsync = promisify(execFile);
const AUTHORIZATION_URL = "https://app.todoist.com/oauth/authorize";
const TOKEN_URL = "https://api.todoist.com/oauth/access_token";
const REGISTRATION_URL = "https://api.todoist.com/oauth/register";
const REDIRECT_URI = "http://localhost:53682/callback";
const SECRET_SERVICE = "task-planner.todoist.oauth";
const REFRESH_SKEW_MS = 60_000;

const TokenResponseSchema = z.object({
  access_token: z.string().min(1),
  refresh_token: z.string().min(1),
  expires_in: z.number().positive(),
  token_type: z.literal("Bearer"),
});

const StoredTokensSchema = TokenResponseSchema.extend({
  client_id: z.string().min(1),
  expires_at: z.number().positive(),
});

type StoredTokens = z.infer<typeof StoredTokensSchema>;

const base64Url = (value: Buffer) => value.toString("base64url");
const randomValue = () => base64Url(randomBytes(32));
const challengeFor = (verifier: string) =>
  base64Url(createHash("sha256").update(verifier).digest());

const readSecret = async (): Promise<string | null> => {
  try {
    if (process.platform === "darwin") {
      const { stdout } = await execFileAsync("security", [
        "find-generic-password",
        "-a",
        process.env.USER ?? "",
        "-s",
        SECRET_SERVICE,
        "-w",
      ]);
      return stdout.trim() || null;
    }
    const { stdout } = await execFileAsync("secret-tool", [
      "lookup",
      "service",
      SECRET_SERVICE,
      "account",
      process.env.USER ?? "",
    ]);
    return stdout.trim() || null;
  } catch {
    return null;
  }
};

const writeSecret = async (secret: string): Promise<void> => {
  if (process.platform === "darwin") {
    await execFileAsync("security", [
      "add-generic-password",
      "-a",
      process.env.USER ?? "",
      "-s",
      SECRET_SERVICE,
      "-w",
      secret,
      "-U",
    ]);
    return;
  }
  await new Promise<void>((resolve, reject) => {
    const child = execFile(
      "secret-tool",
      [
        "store",
        "--label=task-planner Todoist OAuth",
        "service",
        SECRET_SERVICE,
        "account",
        process.env.USER ?? "",
      ],
      (error) => (error ? reject(error) : resolve()),
    );
    child.stdin?.end(secret);
  });
};

const deleteSecret = async (): Promise<void> => {
  if (process.platform === "darwin") {
    await execFileAsync("security", [
      "delete-generic-password",
      "-a",
      process.env.USER ?? "",
      "-s",
      SECRET_SERVICE,
    ]);
    return;
  }
  await execFileAsync("secret-tool", [
    "clear",
    "service",
    SECRET_SERVICE,
    "account",
    process.env.USER ?? "",
  ]);
};

const storeTokens = async (
  tokens: z.infer<typeof TokenResponseSchema>,
  clientId: string,
): Promise<StoredTokens> => {
  const stored: StoredTokens = {
    ...tokens,
    client_id: clientId,
    expires_at: Date.now() + tokens.expires_in * 1000,
  };
  await writeSecret(JSON.stringify(stored));
  return stored;
};

const requestTokens = async (params: URLSearchParams) => {
  const response = await fetch(TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params,
  });
  if (!response.ok)
    throw new Error(
      `Todoist OAuth token request failed: ${response.status} ${await response.text()}`,
    );
  return TokenResponseSchema.parse(await response.json());
};

const registerClient = async (): Promise<string> => {
  const response = await fetch(REGISTRATION_URL, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      client_name: "task-planner",
      redirect_uris: [REDIRECT_URI],
      scope: "data:read_write",
      grant_types: ["authorization_code", "refresh_token"],
      response_types: ["code"],
      token_endpoint_auth_method: "none",
    }),
  });
  if (!response.ok)
    throw new Error(
      `Todoist client registration failed: ${response.status} ${await response.text()}`,
    );
  return z.object({ client_id: z.string().min(1) }).parse(await response.json()).client_id;
};

const openBrowser = async (url: string): Promise<void> => {
  const command = process.platform === "darwin" ? "open" : "xdg-open";
  await execFileAsync(command, [url]);
};

const waitForCallback = (state: string): Promise<string> =>
  new Promise((resolve, reject) => {
    const server = createServer((request, response) => {
      const callback = new URL(request.url ?? "/", REDIRECT_URI);
      const code = callback.searchParams.get("code");
      const returnedState = callback.searchParams.get("state");
      const error = callback.searchParams.get("error");
      const finish = (message: string) => {
        response.writeHead(200, { "Content-Type": "text/html" });
        response.end(`<p>${message}</p>`);
        server.close();
      };
      if (error) {
        finish("Authorization was cancelled. You may close this tab.");
        reject(new Error(`Todoist authorization failed: ${error}`));
        return;
      }
      if (!code || returnedState !== state) {
        finish("Authorization could not be verified. You may close this tab.");
        reject(new Error("Todoist OAuth callback state did not match."));
        return;
      }
      finish("Todoist connected. You may close this tab.");
      resolve(code);
    });
    server.once("error", reject);
    server.listen(53682, "127.0.0.1");
  });

export const login = async (): Promise<void> => {
  const clientId = await registerClient();
  const state = randomValue();
  const verifier = randomValue();
  const url = new URL(AUTHORIZATION_URL);
  url.search = new URLSearchParams({
    client_id: clientId,
    redirect_uri: REDIRECT_URI,
    response_type: "code",
    scope: "data:read_write",
    state,
    code_challenge: challengeFor(verifier),
    code_challenge_method: "S256",
  }).toString();
  const callback = waitForCallback(state);
  await openBrowser(url.toString());
  const code = await callback;
  const tokens = await requestTokens(
    new URLSearchParams({
      grant_type: "authorization_code",
      client_id: clientId,
      code,
      code_verifier: verifier,
      redirect_uri: REDIRECT_URI,
    }),
  );
  await storeTokens(tokens, clientId);
  console.log("Todoist OAuth connection saved to your OS credential store.");
};

export const logout = async (): Promise<void> => {
  await deleteSecret();
  console.log("Todoist OAuth connection removed from your OS credential store.");
};

export const getAccessToken = async (): Promise<string> => {
  const secret = await readSecret();
  let value: unknown = null;
  try {
    value = secret ? JSON.parse(secret) : null;
  } catch {
    value = null;
  }
  const parsed = StoredTokensSchema.safeParse(value);
  if (!parsed.success)
    throw new Error("Todoist is not connected. Run `pnpm auth login` interactively first.");
  if (parsed.data.expires_at - REFRESH_SKEW_MS > Date.now()) return parsed.data.access_token;
  const tokens = await requestTokens(
    new URLSearchParams({
      grant_type: "refresh_token",
      client_id: parsed.data.client_id,
      refresh_token: parsed.data.refresh_token,
    }),
  );
  return (await storeTokens(tokens, parsed.data.client_id)).access_token;
};

export const listProjects = async (): Promise<void> => {
  const response = await fetch("https://api.todoist.com/api/v1/projects", {
    headers: { Authorization: `Bearer ${await getAccessToken()}` },
  });
  if (!response.ok)
    throw new Error(`Todoist project request failed: ${response.status} ${await response.text()}`);
  console.log(JSON.stringify(await response.json(), null, 2));
};
