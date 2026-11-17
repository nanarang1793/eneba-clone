import { useEffect, useState } from "react";
import { Link } from "react-router";
import type { Banner } from "../../api/types";

const ROTATE_MS = 6000;

// No banner art in the seed (we don't copy Eneba's images), so each slide
// gets one of these gradients.
const GRADIENTS = [
  "linear-gradient(120deg, #6d28d9 0%, #a182e8 55%, #fd5a20 130%)",
  "linear-gradient(120deg, #1f0a4d 0%, #7c3aed 60%, #ffd02f 140%)",
  "linear-gradient(120deg, #4c1d95 0%, #2fbf71 120%)",
];

/** Big rotating banner at the top of the homepage. Pauses while hovered. */
export function BannerCarousel({ banners }: { banners: Banner[] }) {
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    if (paused || banners.length < 2) return;
    const timer = setInterval(() => setIndex((i) => (i + 1) % banners.length), ROTATE_MS);
    return () => clearInterval(timer);
  }, [paused, banners.length]);

  if (banners.length === 0) return null;
  const b = banners[index % banners.length];

  return (
    <section
      aria-roledescription="carousel"
      aria-label="Featured"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      className="relative overflow-hidden rounded-xl"
    >
      <div
        className="flex min-h-64 flex-col justify-center gap-4 px-8 py-12 md:min-h-80 md:px-14"
        style={{ background: b.image_url ? `center/cover url(${b.image_url})` : GRADIENTS[index % GRADIENTS.length] }}
      >
        <h2 className="max-w-xl text-3xl font-extrabold leading-tight drop-shadow md:text-5xl">{b.title}</h2>
        {b.subtitle && <p className="max-w-lg text-lg text-white/90">{b.subtitle}</p>}
        <Link
          to={b.cta_url}
          className="w-fit rounded-md bg-cta px-6 py-3 font-semibold text-page shadow-lg hover:bg-cta-hover"
        >
          {b.cta_label}
        </Link>
      </div>

      {banners.length > 1 && (
        <div className="absolute bottom-4 right-6 flex gap-2">
          {banners.map((banner, i) => (
            <button
              key={banner.id}
              onClick={() => setIndex(i)}
              aria-label={`Show banner ${i + 1}: ${banner.title}`}
              aria-current={i === index}
              className={`h-2.5 rounded-full transition-all ${i === index ? "w-8 bg-white" : "w-2.5 bg-white/50 hover:bg-white/80"}`}
            />
          ))}
        </div>
      )}
    </section>
  );
}
