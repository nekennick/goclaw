import { CONTRACT_VERSION } from "../contracts/types.js";
import { ERROR_CODES, createErrorEnvelope } from "../contracts/error-codes.js";
import { PAGE_TEXT_MAX_LENGTH } from "../contracts/tool-schemas.js";
import { searchDocumentContent } from "../search/vietnamese-search.js";
import { VERIFIED_INVENTORY } from "../policy/verified-inventory.js";
export class WarehouseKbService {
    deps;
    constructor(deps) {
        this.deps = deps;
    }
    async search(query, limit = 5) {
        const checkedAt = new Date().toISOString();
        // 1. Fetch current registered documents from GoClaw Vault
        let vaultDocs;
        try {
            vaultDocs = await this.deps.vaultClient.listAllDocuments();
        }
        catch (err) {
            return createErrorEnvelope(ERROR_CODES.SOURCE_UNAVAILABLE, `Không thể kết nối danh mục GoClaw Vault: ${err instanceof Error ? err.message : String(err)}`);
        }
        // 2. Filter documents permitted by PolicyEngine
        const eligibleDocs = [];
        for (const doc of vaultDocs) {
            const check = this.deps.policyEngine.verify(doc);
            if (check.allowed) {
                eligibleDocs.push({ meta: doc, item: check.item });
            }
        }
        if (eligibleDocs.length === 0) {
            return createErrorEnvelope(ERROR_CODES.SOURCE_NOT_READY, "Chưa có tài liệu quy trình nào thỏa mãn chính sách hiệu lực được duyệt.");
        }
        // 3. Read physical files and verify hashes
        const matchCandidates = [];
        for (const { meta, item } of eligibleDocs) {
            let fileRes;
            try {
                fileRes = await this.deps.fileReader.readVerifiedFile(item.root_key, item.relative_path);
            }
            catch (err) {
                return createErrorEnvelope(ERROR_CODES.SOURCE_UNAVAILABLE, `Không đọc được file nguồn ${item.logical_path}: ${err instanceof Error ? err.message : String(err)}`);
            }
            // Check SHA-256 byte freshness
            if (fileRes.contentHash.toLowerCase() !== meta.content_hash.toLowerCase()) {
                return createErrorEnvelope(ERROR_CODES.SOURCE_NOT_READY, `Tài liệu ${item.logical_path} có nội dung thực tế chưa đồng bộ với Vault hash (file hash khác registry).`);
            }
            // Search inside file content
            const match = searchDocumentContent(fileRes.text, query);
            if (match) {
                matchCandidates.push({
                    item: {
                        doc_id: meta.id,
                        title: meta.title || item.title,
                        logical_path: item.logical_path,
                        content_hash: meta.content_hash,
                        updated_at: meta.updated_at,
                        line_start: match.lineStart,
                        line_end: match.lineEnd,
                        snippet: match.snippet,
                    },
                    score: match.score,
                });
            }
        }
        // Sort by score descending and take limit
        matchCandidates.sort((a, b) => b.score - a.score);
        const topResults = matchCandidates.slice(0, limit).map((c) => c.item);
        return {
            ok: true,
            contract_version: CONTRACT_VERSION,
            source_checked_at: checkedAt,
            results: topResults,
        };
    }
    async read(doc_id, expected_hash, cursor) {
        const checkedAt = new Date().toISOString();
        // 1. Check verified inventory allowlist
        const item = VERIFIED_INVENTORY.get(doc_id);
        if (!item) {
            return createErrorEnvelope(ERROR_CODES.DOC_NOT_ALLOWED, `Tài liệu ${doc_id} không thuộc danh mục tri thức kho được duyệt.`);
        }
        // 2. Fetch Vault metadata before reading
        let metaBefore;
        try {
            metaBefore = await this.deps.vaultClient.getDocument(doc_id);
        }
        catch (err) {
            return createErrorEnvelope(ERROR_CODES.SOURCE_UNAVAILABLE, `Lỗi khi kiểm tra metadata tài liệu ${doc_id}: ${err instanceof Error ? err.message : String(err)}`);
        }
        if (!metaBefore) {
            return createErrorEnvelope(ERROR_CODES.DOC_NOT_ALLOWED, `Tài liệu ${doc_id} không tồn tại trong Vault GoClaw.`);
        }
        // Check policy before read
        const checkBefore = this.deps.policyEngine.verify(metaBefore);
        if (!checkBefore.allowed) {
            return createErrorEnvelope(ERROR_CODES.DOC_NOT_ALLOWED, `Tài liệu ${doc_id} không thỏa mãn chính sách truy cập: ${checkBefore.reason}`);
        }
        // Check expected hash
        if (metaBefore.content_hash.toLowerCase() !== expected_hash.toLowerCase()) {
            return createErrorEnvelope(ERROR_CODES.SOURCE_CHANGED, `Nguồn tài liệu ${doc_id} đã thay đổi: expected_hash khác hash hiện hành trong Vault.`);
        }
        // 3. Read physical file bytes
        let fileRes;
        try {
            fileRes = await this.deps.fileReader.readVerifiedFile(item.root_key, item.relative_path);
        }
        catch (err) {
            return createErrorEnvelope(ERROR_CODES.SOURCE_UNAVAILABLE, `Không thể đọc file nguồn ${item.logical_path}: ${err instanceof Error ? err.message : String(err)}`);
        }
        // Verify raw file hash
        if (fileRes.contentHash.toLowerCase() !== expected_hash.toLowerCase()) {
            return createErrorEnvelope(ERROR_CODES.SOURCE_NOT_READY, `Nội dung đĩa của tài liệu ${item.logical_path} không khớp với hash mong đợi.`);
        }
        // 4. Double check metadata after reading
        let metaAfter;
        try {
            metaAfter = await this.deps.vaultClient.getDocument(doc_id);
        }
        catch {
            metaAfter = null;
        }
        if (!metaAfter || metaAfter.content_hash.toLowerCase() !== expected_hash.toLowerCase()) {
            return createErrorEnvelope(ERROR_CODES.SOURCE_CHANGED, `Nguồn tài liệu ${doc_id} đã thay đổi trong quá trình đọc.`);
        }
        // 5. Handle pagination
        let startLine = 1;
        if (cursor) {
            const decoded = this.deps.cursorCodec.decode(cursor, doc_id, expected_hash);
            if (!decoded) {
                return createErrorEnvelope(ERROR_CODES.INVALID_INPUT, "Cursor không hợp lệ, bị giả mạo hoặc không khớp với phiên bản tài liệu hiện tại.");
            }
            startLine = decoded.line_offset;
        }
        const allLines = fileRes.text.split("\n");
        if (startLine > allLines.length) {
            return createErrorEnvelope(ERROR_CODES.INVALID_INPUT, "Cursor vượt quá độ dài của tài liệu.");
        }
        const pageLines = [];
        let currentChars = 0;
        let endLine = startLine;
        for (let i = startLine - 1; i < allLines.length; i++) {
            const line = allLines[i];
            if (currentChars + line.length > PAGE_TEXT_MAX_LENGTH && pageLines.length > 0) {
                break;
            }
            pageLines.push(line);
            currentChars += line.length + 1;
            endLine = i + 1; // 1-based
        }
        const pageText = pageLines.join("\n");
        const hasMore = endLine < allLines.length;
        let nextCursor = null;
        if (hasMore) {
            nextCursor = this.deps.cursorCodec.encode({
                doc_id,
                expected_hash,
                line_offset: endLine + 1,
            });
        }
        return {
            ok: true,
            contract_version: CONTRACT_VERSION,
            source_checked_at: checkedAt,
            doc_id,
            title: metaBefore.title || item.title,
            logical_path: item.logical_path,
            content_hash: metaBefore.content_hash,
            updated_at: metaBefore.updated_at,
            line_start: startLine,
            line_end: endLine,
            text: pageText,
            has_more: hasMore,
            next_cursor: nextCursor,
        };
    }
}
