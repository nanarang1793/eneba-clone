import { apiGet } from "./client";
import type { Block, BlockInfo } from "./types";

export const getHomeLayout = () => apiGet<{ blocks: BlockInfo[] }>("/home/layout").then((r) => r.blocks);

export const getHomeBlock = (id: number) => apiGet<Block>(`/home/blocks/${id}`);
