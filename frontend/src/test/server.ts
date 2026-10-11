import { setupServer } from "msw/node";

/* Each test states the answers it needs with server.use(). A request no test
   expected is an error (see setup.ts). */
export const server = setupServer();
