import test from "node:test";
import assert from "node:assert/strict";
import { searchDocumentContent } from "../src/search/vietnamese-search.js";
import { SAMPLE_DOC_00_CONTENT } from "./fixtures/mock-source-files.js";
import { SNIPPET_MAX_LENGTH } from "../src/contracts/tool-schemas.js";

test("vietnamese-search - finds exact and contextual matches in Vietnamese text", () => {
  const matchKiemKe = searchDocumentContent(SAMPLE_DOC_00_CONTENT, "kiểm kê");
  assert.ok(matchKiemKe);
  assert.ok(matchKiemKe.snippet.includes("Thứ 4"));
  assert.ok(matchKiemKe.snippet.includes("13:30"));
  assert.ok(matchKiemKe.lineStart > 0);
  assert.ok(matchKiemKe.lineEnd >= matchKiemKe.lineStart);

  const match16h = searchDocumentContent(SAMPLE_DOC_00_CONTENT, "16:00");
  assert.ok(match16h);
  assert.ok(match16h.snippet.includes("hạn chót"));
  assert.ok(match16h.snippet.includes("hóa đơn"));

  const matchNhanSu = searchDocumentContent(SAMPLE_DOC_00_CONTENT, "Anh Khoa");
  assert.ok(matchNhanSu);
  assert.ok(matchNhanSu.snippet.includes("Giám đốc điều hành"));
});

test("vietnamese-search - returns null for non-matching query", () => {
  const noMatch = searchDocumentContent(SAMPLE_DOC_00_CONTENT, "xe máy điện vinfast");
  assert.equal(noMatch, null);

  const emptyMatch = searchDocumentContent(SAMPLE_DOC_00_CONTENT, "   ");
  assert.equal(emptyMatch, null);
});

test("vietnamese-search - caps snippet size within contract limit", () => {
  const longText = "# Heading\n" + "Dòng nội dung lặp lại rất dài ".repeat(100) + "\nTừ khóa tìm kiếm nằm ở đây";
  const match = searchDocumentContent(longText, "Từ khóa tìm kiếm");
  assert.ok(match);
  assert.ok(match.snippet.length <= SNIPPET_MAX_LENGTH + 10);
});
