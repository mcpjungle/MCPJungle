import type { ReactNode } from "react";

export function SectionCard({
  title,
  subtitle,
  action,
  children,
}: {
  title: string;
  subtitle?: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="panel">
      <div className="panel-header">
        <div>
          <p className="panel-label">{title}</p>
          {subtitle ? <h3>{subtitle}</h3> : null}
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}
