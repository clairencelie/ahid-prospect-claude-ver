import type { Band } from "../types";

const COLORS: Record<Band, { bg: string; fg: string }> = {
  PASS: { bg: "#e6f4ea", fg: "#1e7e34" },
  REVIEW: { bg: "#fff4e0", fg: "#b8860b" },
  BLOCK: { bg: "#fde8e8", fg: "#c0392b" },
};

export function BandBadge({ band }: { band: Band }) {
  const { bg, fg } = COLORS[band];
  return (
    <span
      style={{
        background: bg,
        color: fg,
        padding: "2px 10px",
        borderRadius: 12,
        fontSize: 12,
        fontWeight: 600,
        letterSpacing: 0.5,
      }}
    >
      {band}
    </span>
  );
}
