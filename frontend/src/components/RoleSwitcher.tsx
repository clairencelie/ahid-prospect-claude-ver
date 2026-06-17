import { useRole } from "../context/RoleContext";

export function RoleSwitcher() {
  const { users, currentUser, setCurrentUser } = useRole();

  return (
    <select
      value={currentUser?.id ?? ""}
      onChange={(e) => {
        const user = users.find((u) => u.id === e.target.value);
        if (user) setCurrentUser(user);
      }}
      style={{ padding: "6px 10px", borderRadius: 6, border: "1px solid #ccc" }}
    >
      {users.map((u) => (
        <option key={u.id} value={u.id}>
          {u.name} ({u.role})
        </option>
      ))}
    </select>
  );
}
