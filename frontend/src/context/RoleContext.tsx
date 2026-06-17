import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { api, setCurrentUserId } from "../api/client";
import type { User } from "../types";

const STORAGE_KEY = "ahid_demo_user_id";

interface RoleContextValue {
  users: User[];
  currentUser: User | null;
  loading: boolean;
  error: string | null;
  setCurrentUser: (user: User) => void;
}

const RoleContext = createContext<RoleContextValue | null>(null);

export function RoleProvider({ children }: { children: ReactNode }) {
  const [users, setUsers] = useState<User[]>([]);
  const [currentUser, setCurrentUserState] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .listUsers()
      .then((list) => {
        setUsers(list);
        const savedId = localStorage.getItem(STORAGE_KEY);
        const initial = list.find((u) => u.id === savedId) ?? list[0] ?? null;
        if (initial) {
          setCurrentUserId(initial.id);
          setCurrentUserState(initial);
        }
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  function setCurrentUser(user: User) {
    setCurrentUserId(user.id);
    setCurrentUserState(user);
    localStorage.setItem(STORAGE_KEY, user.id);
  }

  return (
    <RoleContext.Provider value={{ users, currentUser, loading, error, setCurrentUser }}>
      {children}
    </RoleContext.Provider>
  );
}

export function useRole(): RoleContextValue {
  const ctx = useContext(RoleContext);
  if (!ctx) throw new Error("useRole must be used within a RoleProvider");
  return ctx;
}
