import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { apiUrl, Config } from "@/base/config";
import { Button } from "@/components/ui/button";
import ErrorHandler from "@/utils/ErrorHandler";
import { debugError } from "@/utils/Logger";

const TAG = "HomePage";

type ApiEnvelope<T> = {
  code: string;
  message: string;
  result?: T;
};

type HealthResult = {
  service: string;
  version: string;
  environment: string;
};

type ProbeState =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "up"; health: HealthResult }
  | { status: "down"; reason: string };

/** Landing page. Confirms the browser can reach the Go API. */
export default function HomePage() {
  const [probe, setProbe] = useState<ProbeState>({ status: "idle" });

  const checkHealth = useCallback(async () => {
    setProbe({ status: "loading" });
    try {
      const response = await fetch(apiUrl("/public/v1/health"));
      const body: ApiEnvelope<HealthResult> = await response.json();

      if (!response.ok || !body.result) {
        setProbe({
          status: "down",
          reason: body.message || response.statusText,
        });
        return;
      }

      setProbe({ status: "up", health: body.result });
    } catch (error) {
      debugError(TAG, error);
      setProbe({
        status: "down",
        reason: error instanceof Error ? error.message : "unreachable",
      });
    }
  }, []);

  useEffect(() => {
    void checkHealth();
  }, [checkHealth]);

  return (
    <ErrorHandler componentName={TAG}>
      <div className="flex min-h-screen flex-1 flex-col">
        <header className="flex h-16 shrink-0 items-center border-b px-6">
          <h1 className="font-semibold text-lg">{Config.appName}</h1>
        </header>

        <main className="flex flex-1 flex-col items-center justify-center gap-8 p-6 text-center">
          <div className="space-y-2">
            <h2 className="font-bold text-2xl">LinkedIn Profile API</h2>
            <p className="max-w-md text-muted-foreground">
              Submit a LinkedIn profile URL and get the profile back as
              structured JSON.
            </p>
          </div>

          <ApiStatus probe={probe} />

          <div className="flex gap-3">
            <Button
              onClick={() => {
                void checkHealth();
                toast.info("Re-checking the API…");
              }}
            >
              Re-check API
            </Button>
          </div>
        </main>
      </div>
    </ErrorHandler>
  );
}

function ApiStatus({ probe }: { probe: ProbeState }) {
  switch (probe.status) {
    case "idle":
    case "loading":
      return <p className="text-muted-foreground text-sm">Checking the API…</p>;

    case "up":
      return (
        <p className="text-sm">
          API is <span className="font-medium text-primary">up</span> —{" "}
          {probe.health.service} v{probe.health.version} (
          {probe.health.environment})
        </p>
      );

    case "down":
      return (
        <p className="text-destructive text-sm">
          API is unreachable: {probe.reason}
        </p>
      );
  }
}
