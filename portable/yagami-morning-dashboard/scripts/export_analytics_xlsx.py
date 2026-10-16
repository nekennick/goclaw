#!/usr/bin/env python3
"""Write the current analytics snapshot as a dependency-free XLSX workbook."""
from __future__ import annotations

import json
import zipfile
from pathlib import Path
from xml.sax.saxutils import escape

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "data/analytics/analytics.json"
OUT = ROOT / "outputs/bao-cao-mat-hang-ngung-mua-hoa-don-2026.xlsx"


def col(n):
    result = ""
    while n:
        n, rem = divmod(n - 1, 26)
        result = chr(65 + rem) + result
    return result


def cell(row, column, value):
    ref = f"{col(column)}{row}"
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        return f'<c r="{ref}"><v>{value}</v></c>'
    text = escape("" if value is None else str(value)).replace("\n", "&#10;")
    return f'<c r="{ref}" t="inlineStr"><is><t xml:space="preserve">{text}</t></is></c>'


def sheet_xml(rows, widths=None):
    body = []
    for row_number, values in enumerate(rows, 1):
        body.append(f'<row r="{row_number}">' + "".join(cell(row_number, i, value) for i, value in enumerate(values, 1)) + "</row>")
    last_col = col(max((len(row) for row in rows), default=1))
    last_row = len(rows)
    cols = ""
    if widths:
        cols = "<cols>" + "".join(f'<col min="{i}" max="{i}" width="{width}" customWidth="1"/>' for i, width in enumerate(widths, 1)) + "</cols>"
    return f'''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><dimension ref="A1:{last_col}{last_row}"/>{cols}<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/><selection pane="bottomLeft" activeCell="A2" sqref="A2"/></sheetView></sheetViews><sheetData>{"".join(body)}</sheetData><autoFilter ref="A1:{last_col}{last_row}"/></worksheet>'''


def main():
    snapshot = json.loads(DATA.read_text(encoding="utf-8"))
    report = snapshot["stoppedProducts"]
    rows = [["Khách hàng / chi nhánh", "Mặt hàng", "Đơn từ đầu năm", "Hóa đơn gần nhất", "Ngày mua gần nhất", "Đã cách (ngày)"]]
    rows += [[x["branch"], x["product"], x["baselineOrders"], x["latestInvoice"], x["latestPurchaseDate"], x["daysSincePurchase"]] for x in report.get("items", [])]
    alias_rows = [["Mã đang hoạt động", "Mã cũ", "Tên sản phẩm"]]
    for item in report.get("productAliases", []):
        for old in item.get("retiredCodes", []):
            alias_rows.append([item.get("activeCode"), old, item.get("name")])
    excluded_rows = [["Mã bị loại", "Lý do"]] + [[code, "Mã sản phẩm inactive"] for code in report.get("ignoredInactiveCodes", [])] + [["{DEL}", "Mã đã xóa / đánh dấu xóa"]]
    summary_rows = [["BÁO CÁO MẶT HÀNG KHÁCH HÀNG NGỪNG MUA"], ["Nguồn", report.get("source")], ["Dữ liệu từ", report.get("dataFrom")], ["Dữ liệu đến", report.get("dataTo")], ["Giai đoạn nền", f'{report["baseline"]["from"]} đến {report["baseline"]["to"]}'], ["30 ngày gần nhất", f'{report["recent"]["from"]} đến {report["recent"]["to"]}'], ["Số hóa đơn đã đọc", report.get("rows")], ["Số trường hợp cần kiểm tra", report.get("stoppedCount")], ["Quy tắc", "Tối thiểu 2 hóa đơn từ đầu năm và không có hóa đơn trong 30 ngày gần nhất"], ["Chuẩn hóa", "Gom theo customerId; mã cũ có mã hoạt động thay thế được gom về mã mới; mã inactive và {DEL} bị loại"]]
    sheets = [("Kết quả phân tích", rows, [28, 48, 16, 18, 16, 15]), ("Mã hàng đã gom", alias_rows, [22, 18, 48]), ("Mã hàng bị loại", excluded_rows, [24, 34]), ("Thông tin báo cáo", summary_rows, [28, 100])]
    OUT.parent.mkdir(parents=True, exist_ok=True)
    content_types = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>' + ''.join(f'<Override PartName="/xl/worksheets/sheet{i}.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>' for i in range(1, len(sheets) + 1)) + '</Types>'
    workbook = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>' + ''.join(f'<sheet name="{escape(name)}" sheetId="{i}" r:id="rId{i}"/>' for i, (name, _, _) in enumerate(sheets, 1)) + '</sheets></workbook>'
    rels = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>'
    workbook_rels = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">' + ''.join(f'<Relationship Id="rId{i}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet{i}.xml"/>' for i in range(1, len(sheets) + 1)) + '<Relationship Id="rId99" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>'
    styles = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font><sz val="10"/><name val="Arial"/></font></fonts><fills count="1"><fill><patternFill patternType="none"/></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellXfs></styleSheet>'
    with zipfile.ZipFile(OUT, "w", zipfile.ZIP_DEFLATED) as book:
        book.writestr("[Content_Types].xml", content_types)
        book.writestr("_rels/.rels", rels)
        book.writestr("xl/workbook.xml", workbook)
        book.writestr("xl/_rels/workbook.xml.rels", workbook_rels)
        book.writestr("xl/styles.xml", styles)
        for i, (_, data, widths) in enumerate(sheets, 1):
            book.writestr(f"xl/worksheets/sheet{i}.xml", sheet_xml(data, widths))
    print(json.dumps({"ok": True, "file": str(OUT), "rows": len(rows) - 1}, ensure_ascii=False))


if __name__ == "__main__":
    main()
