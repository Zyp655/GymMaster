export function SectionHeading({
  eyebrow,
  title,
  subtitle,
}: {
  eyebrow: string;
  title: string;
  subtitle?: string;
}) {
  return (
    <div className="grid max-w-2xl gap-2">
      <span className="text-caption font-semibold tracking-wide text-brand-600 uppercase">
        {eyebrow}
      </span>
      <h2 className="text-h1 font-semibold text-balance text-primary">{title}</h2>
      {subtitle ? <p className="text-body-lg text-secondary">{subtitle}</p> : null}
    </div>
  );
}
