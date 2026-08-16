import type { Brain, Page } from "@/entities/brain/entity";

export type Visibility = "all" | Brain["visibility"];
export type BrainMetrics = { pages: Page[] | null };
