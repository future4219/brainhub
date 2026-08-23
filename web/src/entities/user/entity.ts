import type { Brain } from "@/entities/brain/entity";

export type User = {
  id: string;
  email: string;
  name: string;
  state: string;
  created_at: string;
  updated_at: string;
};

export type ViewerState = {
  viewer: User | null | undefined;
  brains: Brain[] | null;
  sessionUnavailable: boolean;
  onLogout: () => void;
};
