/**
 * Product cover. The seed has no cover art (we don't copy Eneba's images),
 * so when image_url is empty we draw a gradient whose colour comes from the
 * title: every variant of "Elden Ring" gets the same colour on every page.
 */
export function Cover({
  title,
  imageUrl,
  showTitle = true,
  className = "",
}: {
  title: string;
  imageUrl: string | null;
  showTitle?: boolean; // tiny thumbnails (search dropdown) hide the text
  className?: string;
}) {
  if (imageUrl) {
    return <img src={imageUrl} alt="" loading="lazy" className={`object-cover ${className}`} />;
  }

  let hue = 0;
  for (const ch of title) hue = (hue * 31 + ch.charCodeAt(0)) % 360;

  return (
    <div
      className={`flex items-center justify-center p-3 text-center ${className}`}
      style={{ background: `linear-gradient(160deg, hsl(${hue} 60% 45%), hsl(${(hue + 60) % 360} 55% 22%))` }}
    >
      {showTitle && <span className="font-extrabold leading-tight drop-shadow-md">{title}</span>}
    </div>
  );
}
