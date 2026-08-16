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
  sessionUnavailable: boolean;
  onLogout: () => void;
};
