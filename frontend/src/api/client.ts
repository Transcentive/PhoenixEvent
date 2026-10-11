/* The API is always same-origin: Vite proxies /api/v1 in development and
   nginx does on the droplet, so no address is configured here. */

export interface Health {
  status: "ok" | "unavailable";
  database: "ok" | "unreachable";
  build: string;
}

export async function getHealth(): Promise<Health> {
  const response = await fetch("/api/v1/health", {
    headers: { Accept: "application/json" },
  });
  /* 503 carries the same body as 200, saying what is unavailable. Anything
     else means the API itself did not answer. */
  if (response.ok || response.status === 503) {
    return (await response.json()) as Health;
  }
  throw new Error(`The API answered ${response.status}`);
}
