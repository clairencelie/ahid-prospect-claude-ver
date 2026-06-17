import { NavLink, Outlet } from "react-router-dom";
import { RoleSwitcher } from "./RoleSwitcher";
import { useRole } from "../context/RoleContext";

const navItemStyle = ({ isActive }: { isActive: boolean }): React.CSSProperties => ({
  padding: "8px 12px",
  borderRadius: 6,
  textDecoration: "none",
  color: isActive ? "#fff" : "#333",
  background: isActive ? "#2d6cdf" : "transparent",
  fontSize: 14,
});

export function Layout() {
  const { currentUser, loading, error } = useRole();

  if (loading) return <div style={{ padding: 24 }}>Loading...</div>;
  if (error) return <div style={{ padding: 24, color: "#c0392b" }}>Failed to load: {error}</div>;

  const role = currentUser?.role;
  const canSeeReviewQueue = role === "branch_co" || role === "compliance" || role === "audit" || role === "admin";
  const canSeeAudit = role === "audit" || role === "admin";

  return (
    <div style={{ fontFamily: "system-ui, sans-serif", color: "#222" }}>
      <header
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          padding: "12px 24px",
          borderBottom: "1px solid #e0e0e0",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 24 }}>
          <strong>AHID Prospect Protection</strong>
          <nav style={{ display: "flex", gap: 4 }}>
            <NavLink to="/" style={navItemStyle} end>
              Intake
            </NavLink>
            <NavLink to="/prospects" style={navItemStyle}>
              Prospects
            </NavLink>
            {canSeeReviewQueue && (
              <NavLink to="/review-queue" style={navItemStyle}>
                Review Queue
              </NavLink>
            )}
            <NavLink to="/companies" style={navItemStyle}>
              Company Master
            </NavLink>
            <NavLink to="/uw" style={navItemStyle}>
              UW
            </NavLink>
            <NavLink to="/monitoring" style={navItemStyle}>
              Monitoring
            </NavLink>
            {canSeeAudit && (
              <NavLink to="/audit-log" style={navItemStyle}>
                Audit Log
              </NavLink>
            )}
          </nav>
        </div>
        <RoleSwitcher />
      </header>
      <main style={{ padding: 24, maxWidth: 1100, margin: "0 auto" }}>
        <Outlet />
      </main>
    </div>
  );
}
