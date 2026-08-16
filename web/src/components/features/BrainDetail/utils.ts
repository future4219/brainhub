export function copyLabel(state?: "copied" | "failed") {
  return state === "copied" ? "コピー済み" : "コピー";
}
