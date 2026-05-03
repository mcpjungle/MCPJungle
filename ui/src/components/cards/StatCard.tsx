import { Link } from "react-router-dom";

type StatCardProps = {
  label: string;
  value: string | number;
  tone?: "default" | "accent" | "up" | "down";
  detail?: string;
  linkTo?: string;
};

export function StatCard({ label, value, tone = "default", detail, linkTo }: StatCardProps) {
  const color =
    tone === "accent" ? "text-accent" : tone === "up" ? "text-up" : tone === "down" ? "text-down" : "text-body";

  const inner = (
    <>
      <p className="text-xs uppercase tracking-[0.18em] text-muted">{label}</p>
      <p className={`numeric mt-4 text-4xl font-semibold ${color}`}>{value}</p>
      {detail ? <p className="mt-3 text-sm text-muted">{detail}</p> : null}
      {linkTo ? (
        <p className="mt-3 text-xs text-accent/70 group-hover:text-accent">View →</p>
      ) : null}
    </>
  );

  if (linkTo) {
    return (
      <Link
        to={linkTo}
        className="group rounded-panel border border-line bg-[#121820] p-5 transition hover:border-accent/40 hover:bg-[#121820]"
      >
        {inner}
      </Link>
    );
  }

  return (
    <div className="rounded-panel border border-line bg-[#121820] p-5">
      {inner}
    </div>
  );
}
