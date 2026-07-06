import React from "react";
import ReactDOM from "react-dom/client";
import {
  AlertTriangle,
  CheckCircle2,
  ClipboardList,
  Gauge,
  Loader2,
  RefreshCw,
  Search,
} from "lucide-react";
import "./styles.css";

type ReportCheck = {
  name: string;
  status: string;
  details: string;
};

type SeoReport = {
  id: string;
  url: string;
  title: string;
  summary: string;
  score: number;
  report: {
    checks?: ReportCheck[];
    recommendations?: string[];
  };
  created_at: string;
};

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

type DeviceIdentity = {
  fingerprint: string;
  details: Record<string, string | number | boolean | null>;
};

function getDeviceIdentity(): DeviceIdentity {
  const screenSize = `${window.screen.width}x${window.screen.height}x${window.screen.colorDepth}`;
  const details = {
    userAgent: navigator.userAgent,
    language: navigator.language,
    languages: navigator.languages?.join(",") ?? "",
    platform: navigator.platform,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    screen: screenSize,
    viewport: `${window.innerWidth}x${window.innerHeight}`,
    hardwareConcurrency: navigator.hardwareConcurrency ?? null,
    deviceMemory: "deviceMemory" in navigator ? (navigator as Navigator & { deviceMemory?: number }).deviceMemory ?? null : null,
    touchPoints: navigator.maxTouchPoints ?? 0,
    cookiesEnabled: navigator.cookieEnabled,
  };

  return {
    fingerprint: Object.values(details).join("|"),
    details,
  };
}

function App() {
  const [url, setUrl] = React.useState("");
  const [reports, setReports] = React.useState<SeoReport[]>([]);
  const [isLoading, setIsLoading] = React.useState(true);
  const [isCreating, setIsCreating] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const deviceIdentity = React.useMemo(() => getDeviceIdentity(), []);

  const loadReports = React.useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams({
        device_fingerprint: deviceIdentity.fingerprint,
      });
      const response = await fetch(`${apiBaseUrl}/seo-reports?${params.toString()}`);
      const payload = await response.json();

      if (!response.ok) {
        throw new Error(payload.error ?? "Failed to load reports.");
      }

      setReports(payload);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load reports.");
    } finally {
      setIsLoading(false);
    }
  }, [deviceIdentity.fingerprint]);

  React.useEffect(() => {
    loadReports();
  }, [loadReports]);

  async function createReport(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsCreating(true);
    setError(null);

    try {
      const response = await fetch(`${apiBaseUrl}/seo-reports`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          url,
          device_fingerprint: deviceIdentity.fingerprint,
          device_details: deviceIdentity.details,
        }),
      });
      const payload = await response.json();

      if (!response.ok) {
        throw new Error(payload.error ?? "Failed to generate report.");
      }

      setUrl("");
      setReports((current) => [payload, ...current]);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to generate report.");
    } finally {
      setIsCreating(false);
    }
  }

  return (
    <main className="app-shell">
      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">AI SEO Auditor</p>
            <h1>Audit queue</h1>
          </div>
          <button className="icon-button" type="button" onClick={loadReports} aria-label="Refresh reports">
            <RefreshCw size={18} />
          </button>
        </header>

        <section className="input-panel" aria-label="Create SEO audit">
          <form onSubmit={createReport} className="audit-form">
            <label htmlFor="url">Website URL</label>
            <div className="url-row">
              <Search className="input-icon" size={18} />
              <input
                id="url"
                type="url"
                placeholder="https://example.com"
                value={url}
                onChange={(event) => setUrl(event.target.value)}
                required
              />
              <button type="submit" disabled={isCreating}>
                {isCreating ? <Loader2 className="spin" size={18} /> : <ClipboardList size={18} />}
                <span>{isCreating ? "Auditing" : "Generate"}</span>
              </button>
            </div>
          </form>
          {error ? (
            <div className="notice" role="alert">
              <AlertTriangle size={18} />
              <span>{error}</span>
            </div>
          ) : null}
        </section>

        <section className="report-grid" aria-label="SEO reports">
          {isLoading ? <LoadingState /> : null}
          {!isLoading && reports.length === 0 ? <EmptyState /> : null}
          {!isLoading ? reports.map((report) => <ReportCard key={report.id} report={report} />) : null}
        </section>
      </section>
    </main>
  );
}

function LoadingState() {
  return (
    <div className="state-box">
      <Loader2 className="spin" size={22} />
      <span>Loading reports</span>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="state-box">
      <ClipboardList size={22} />
      <span>No reports yet</span>
    </div>
  );
}

function ReportCard({ report }: { report: SeoReport }) {
  const checks = report.report?.checks ?? [];
  const recommendations = report.report?.recommendations ?? [];

  return (
    <article className="report-card">
      <div className="card-head">
        <div className="title-block">
          <h2>{report.title || report.url}</h2>
          <a href={report.url} target="_blank" rel="noreferrer">
            {report.url}
          </a>
        </div>
        <div className="score-pill">
          <Gauge size={18} />
          <span>{report.score}</span>
        </div>
      </div>

      <p className="summary">{report.summary}</p>

      <div className="checks">
        {checks.map((check) => (
          <div className="check-row" key={check.name}>
            <StatusIcon status={check.status} />
            <div>
              <strong>{check.name}</strong>
              <p>{check.details}</p>
            </div>
          </div>
        ))}
      </div>

      {recommendations.length > 0 ? (
        <div className="recommendations">
          <h3>Recommendations</h3>
          <ul>
            {recommendations.map((recommendation) => (
              <li key={recommendation}>{recommendation}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </article>
  );
}

function StatusIcon({ status }: { status: string }) {
  if (status === "pass") {
    return <CheckCircle2 className="status-pass" size={20} />;
  }

  return <AlertTriangle className="status-warn" size={20} />;
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
