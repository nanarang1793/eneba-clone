import { useQuery } from "@tanstack/react-query";
import { getHomeBlock, getHomeLayout } from "../../api/home";
import type { BlockInfo } from "../../api/types";
import { usePageTitle } from "../../components/usePageTitle";
import { BannerCarousel } from "./BannerCarousel";
import { ProductRow } from "./ProductRow";

/**
 * The homepage is data-driven: the server says which blocks to show and in
 * what order (GET /api/home/layout), and each block then loads its own data
 * (GET /api/home/blocks/{id}). Blocks load in parallel and fail on their own.
 * If one breaks, only that block shows an error.
 */
export function HomePage() {
  usePageTitle(null); // just "Eneba clone"
  const layout = useQuery({ queryKey: ["home-layout"], queryFn: getHomeLayout });

  if (layout.isError) {
    return <BlockError title="Couldn't load the homepage" onRetry={() => layout.refetch()} />;
  }
  if (!layout.data) {
    return (
      <div className="space-y-10">
        <div className="h-64 animate-pulse rounded-xl bg-surface md:h-80" />
        <RowSkeleton />
      </div>
    );
  }

  return (
    <div className="space-y-12">
      {layout.data.map((info) => (
        <HomeBlock key={info.id} info={info} />
      ))}
    </div>
  );
}

/** Loads one block and picks the component for its type. */
function HomeBlock({ info }: { info: BlockInfo }) {
  const block = useQuery({ queryKey: ["home-block", info.id], queryFn: () => getHomeBlock(info.id) });

  if (block.isError) return <BlockError title={`Couldn't load “${info.title}”`} onRetry={() => block.refetch()} />;
  if (!block.data) return info.type === "banner" ? <div className="h-64 animate-pulse rounded-xl bg-surface md:h-80" /> : <RowSkeleton />;

  const b = block.data;
  switch (b.type) {
    case "banner":
      return <BannerCarousel banners={b.data.items} />;
    case "best_sellers":
    case "deals":
    case "new_releases":
    case "by_type":
      return <ProductRow title={b.title} products={b.data.items} seeAllUrl={b.data.see_all_url} />;
    default:
      return null; // a block type this version of the frontend doesn't know yet
  }
}

function RowSkeleton() {
  return (
    <div>
      <div className="mb-4 h-7 w-48 animate-pulse rounded bg-surface" />
      <div className="flex gap-3 overflow-hidden">
        {Array.from({ length: 6 }, (_, i) => (
          <div key={i} className="aspect-[3/5] w-44 shrink-0 animate-pulse rounded-lg bg-surface" />
        ))}
      </div>
    </div>
  );
}

function BlockError({ title, onRetry }: { title: string; onRetry: () => void }) {
  return (
    <div role="alert" className="flex items-center justify-between gap-4 rounded-lg border border-discount/50 bg-discount/10 px-5 py-4">
      <p className="text-sm">{title}</p>
      <button onClick={onRetry} className="rounded-md bg-surface px-3 py-1.5 text-sm hover:bg-surface-hover">
        Try again
      </button>
    </div>
  );
}
