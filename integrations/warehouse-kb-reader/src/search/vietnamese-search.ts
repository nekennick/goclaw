import { SNIPPET_MAX_LENGTH } from "../contracts/tool-schemas.js";

export interface SearchMatch {
  lineStart: number;
  lineEnd: number;
  snippet: string;
  score: number;
}

export function searchDocumentContent(
  text: string,
  query: string
): SearchMatch | null {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) return null;

  const queryTerms = normalizedQuery.split(/\s+/).filter((t) => t.length > 0);
  if (queryTerms.length === 0) return null;

  const lines = text.split("\n");
  let bestMatch: { start: number; end: number; score: number; text: string } | null = null;

  for (let i = 0; i < lines.length; i++) {
    const lineLower = lines[i].toLowerCase();
    let lineScore = 0;

    let hasMatch = false;

    // Exact phrase match gives high boost
    if (lineLower.includes(normalizedQuery)) {
      lineScore += 50;
      hasMatch = true;
    }

    // Keyword match count
    for (const term of queryTerms) {
      if (lineLower.includes(term)) {
        lineScore += 10;
        hasMatch = true;
      }
    }

    // Heading boost (# or ##) only when there is a match in this line
    if (hasMatch && lines[i].trim().startsWith("#")) {
      lineScore += 15;
    }

    if (lineScore > 0) {
      // Create a contextual window around the matching line (up to 4 lines before and after)
      const windowStart = Math.max(0, i - 2);
      const windowEnd = Math.min(lines.length - 1, i + 3);
      const snippetLines = lines.slice(windowStart, windowEnd + 1);
      let snippet = snippetLines.join("\n").trim();

      if (snippet.length > SNIPPET_MAX_LENGTH) {
        snippet = snippet.slice(0, SNIPPET_MAX_LENGTH) + "...";
      }

      if (!bestMatch || lineScore > bestMatch.score) {
        bestMatch = {
          start: windowStart + 1, // 1-based index
          end: windowEnd + 1,
          score: lineScore,
          text: snippet,
        };
      }
    }
  }

  if (!bestMatch) return null;

  return {
    lineStart: bestMatch.start,
    lineEnd: bestMatch.end,
    snippet: bestMatch.text,
    score: bestMatch.score,
  };
}
