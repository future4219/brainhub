import assert from "node:assert/strict";

// Custom font sizes must not replace text colors on buttons and feedback.
const { cn } = await import("./utils.ts");
for (const size of ["ui", "body", "section", "title"]) {
  assert.equal(cn("text-canvas", `text-${size}`), `text-canvas text-${size}`);
  assert.equal(
    cn(`text-${size}`, "text-text-muted"),
    `text-${size} text-text-muted`,
  );
  assert.equal(
    cn("text-canvas", `text-${size}`, "text-sm"),
    "text-canvas text-sm",
  );
}
