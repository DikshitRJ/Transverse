/** Browser MSW worker — started by `MockProvider` (client components / TanStack Query). */
import { setupWorker } from "msw/browser";
import { handlers } from "./handlers";

export const worker = setupWorker(...handlers);
