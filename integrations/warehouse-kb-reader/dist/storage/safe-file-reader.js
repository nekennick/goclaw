import { promises as fs } from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
export const MAX_FILE_SIZE_BYTES = 1024 * 1024; // 1 MiB limit
export const ALLOWED_EXTENSIONS = new Set([".md", ".json", ".csv", ".txt"]);
export class SafeFileReader {
    rootMap;
    constructor(roots) {
        this.rootMap = new Map([
            ["warehouse-knowledge-base", path.resolve(roots.warehouseKbRoot)],
            ["yagami-knowledge-base", path.resolve(roots.yagamiKbRoot)],
        ]);
    }
    /**
     * Resolves, checks boundaries and safely reads a verified relative path within a designated root.
     */
    async readVerifiedFile(rootKey, relativePath) {
        const rootDir = this.rootMap.get(rootKey);
        if (!rootDir) {
            throw new Error(`Unknown rootKey: ${rootKey}`);
        }
        // 1. Strict relative path validation
        if (path.isAbsolute(relativePath)) {
            throw new Error(`Absolute path not allowed: ${relativePath}`);
        }
        const ext = path.extname(relativePath).toLowerCase();
        if (!ALLOWED_EXTENSIONS.has(ext)) {
            throw new Error(`Unsupported extension ${ext} for file: ${relativePath}`);
        }
        // Check path traversal segment by segment
        const normalizedRelative = path.normalize(relativePath).replace(/\\/g, "/");
        if (normalizedRelative.startsWith("../") || normalizedRelative.includes("/../") || normalizedRelative === "..") {
            throw new Error(`Path traversal attempted: ${relativePath}`);
        }
        const fullPath = path.resolve(rootDir, normalizedRelative);
        // Ensure resolved path starts strictly within rootDir directory boundaries
        const relativeToRoot = path.relative(rootDir, fullPath);
        if (relativeToRoot.startsWith("..") || path.isAbsolute(relativeToRoot)) {
            throw new Error(`Path escapes root directory boundary: ${relativePath}`);
        }
        // 2. Check stats and verify it's a regular file, not a symlink or device
        const lstat = await fs.lstat(fullPath);
        if (lstat.isSymbolicLink()) {
            throw new Error(`Symbolic links are strictly prohibited: ${relativePath}`);
        }
        if (!lstat.isFile()) {
            throw new Error(`Target is not a regular file: ${relativePath}`);
        }
        if (lstat.size > MAX_FILE_SIZE_BYTES) {
            throw new Error(`File size ${lstat.size} exceeds maximum allowable limit of ${MAX_FILE_SIZE_BYTES} bytes`);
        }
        // 3. Read raw bytes directly
        const contentRaw = await fs.readFile(fullPath);
        // 4. Calculate SHA-256 on RAW bytes before any normalization/decoding
        const contentHash = createHash("sha256").update(contentRaw).digest("hex");
        // 5. Decode UTF-8 string
        const text = contentRaw.toString("utf-8");
        const lineCount = text.split("\n").length;
        return {
            contentRaw,
            contentHash,
            text,
            lineCount,
        };
    }
}
