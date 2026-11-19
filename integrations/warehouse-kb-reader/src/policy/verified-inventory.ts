export type RootKey = "warehouse-knowledge-base" | "yagami-knowledge-base";

export interface VerifiedDocItem {
  doc_id: string;
  root_key: RootKey;
  relative_path: string;
  logical_path: string;
  title: string;
  expected_scope: "shared";
}

export const VERIFIED_INVENTORY: ReadonlyMap<string, VerifiedDocItem> = new Map([
  // warehouse-knowledge-base (12 documents)
  [
    "01a1900e-ab0b-749c-9808-af69babace6c",
    {
      doc_id: "01a1900e-ab0b-749c-9808-af69babace6c",
      root_key: "warehouse-knowledge-base",
      relative_path: "00-TONG-QUAN-NHAN-SU-VA-NOI-QUY.md",
      logical_path: "warehouse-knowledge-base/00-TONG-QUAN-NHAN-SU-VA-NOI-QUY.md",
      title: "00-TONG-QUAN-NHAN-SU-VA-NOI-QUY",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab0d-7aee-90cc-c66d35860036",
    {
      doc_id: "01a1900e-ab0d-7aee-90cc-c66d35860036",
      root_key: "warehouse-knowledge-base",
      relative_path: "01-TIMELINE-NHIP-VAN-HANH-HANG-NGAY.md",
      logical_path: "warehouse-knowledge-base/01-TIMELINE-NHIP-VAN-HANH-HANG-NGAY.md",
      title: "01-TIMELINE-NHIP-VAN-HANH-HANG-NGAY",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab10-7017-ba03-48b32b9774f0",
    {
      doc_id: "01a1900e-ab10-7017-ba03-48b32b9774f0",
      root_key: "warehouse-knowledge-base",
      relative_path: "02-DONG-GOI-BAO-QUAN-VA-4-MA-SIET-TON.md",
      logical_path: "warehouse-knowledge-base/02-DONG-GOI-BAO-QUAN-VA-4-MA-SIET-TON.md",
      title: "02-DONG-GOI-BAO-QUAN-VA-4-MA-SIET-TON",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab11-7b55-946c-78a0a76e9815",
    {
      doc_id: "01a1900e-ab11-7b55-946c-78a0a76e9815",
      root_key: "warehouse-knowledge-base",
      relative_path: "03-QUY-TRINH-KIOTVIET-CHUYEN-KHO-VA-DOI-TRA.md",
      logical_path: "warehouse-knowledge-base/03-QUY-TRINH-KIOTVIET-CHUYEN-KHO-VA-DOI-TRA.md",
      title: "03-QUY-TRINH-KIOTVIET-CHUYEN-KHO-VA-DOI-TRA",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab15-7ce3-9233-ee6bc04f9ead",
    {
      doc_id: "01a1900e-ab15-7ce3-9233-ee6bc04f9ead",
      root_key: "warehouse-knowledge-base",
      relative_path: "04-CAM-NANG-XU-LY-SU-CO-VA-GIAO-VAN.md",
      logical_path: "warehouse-knowledge-base/04-CAM-NANG-XU-LY-SU-CO-VA-GIAO-VAN.md",
      title: "04-CAM-NANG-XU-LY-SU-CO-VA-GIAO-VAN",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab17-7e8d-b424-cceb86a42fc5",
    {
      doc_id: "01a1900e-ab17-7e8d-b424-cceb86a42fc5",
      root_key: "warehouse-knowledge-base",
      relative_path: "05-BO-CAU-HOI-SAT-HACH-NHAN-VIEN-MOI.md",
      logical_path: "warehouse-knowledge-base/05-BO-CAU-HOI-SAT-HACH-NHAN-VIEN-MOI.md",
      title: "05-BO-CAU-HOI-SAT-HACH-NHAN-VIEN-MOI",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab1a-70b5-98ee-306d0997bc3a",
    {
      doc_id: "01a1900e-ab1a-70b5-98ee-306d0997bc3a",
      root_key: "warehouse-knowledge-base",
      relative_path: "06-QUAN-TRI-DU-LIEU-VA-TU-DIEN.md",
      logical_path: "warehouse-knowledge-base/06-QUAN-TRI-DU-LIEU-VA-TU-DIEN.md",
      title: "06-QUAN-TRI-DU-LIEU-VA-TU-DIEN",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab1b-7cbe-8647-36b6ac65857a",
    {
      doc_id: "01a1900e-ab1b-7cbe-8647-36b6ac65857a",
      root_key: "warehouse-knowledge-base",
      relative_path: "07-DANH-MUC-VAN-DE-CAN-XAC-NHAN.md",
      logical_path: "warehouse-knowledge-base/07-DANH-MUC-VAN-DE-CAN-XAC-NHAN.md",
      title: "07-DANH-MUC-VAN-DE-CAN-XAC-NHAN",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab1f-7bdd-9fcf-9ba994b0b6fb",
    {
      doc_id: "01a1900e-ab1f-7bdd-9fcf-9ba994b0b6fb",
      root_key: "warehouse-knowledge-base",
      relative_path: "08-NHAT-KY-RA-SOAT-VA-CAU-HOI.md",
      logical_path: "warehouse-knowledge-base/08-NHAT-KY-RA-SOAT-VA-CAU-HOI.md",
      title: "08-NHAT-KY-RA-SOAT-VA-CAU-HOI",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab23-76d4-876d-ce77634bf019",
    {
      doc_id: "01a1900e-ab23-76d4-876d-ce77634bf019",
      root_key: "warehouse-knowledge-base",
      relative_path: "MANIFEST.json",
      logical_path: "warehouse-knowledge-base/MANIFEST.json",
      title: "MANIFEST",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab26-7800-a3cf-873511ba7f41",
    {
      doc_id: "01a1900e-ab26-7800-a3cf-873511ba7f41",
      root_key: "warehouse-knowledge-base",
      relative_path: "README.md",
      logical_path: "warehouse-knowledge-base/README.md",
      title: "README",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab29-7731-83fc-f23a184d7860",
    {
      doc_id: "01a1900e-ab29-7731-83fc-f23a184d7860",
      root_key: "warehouse-knowledge-base",
      relative_path: "RULE_REGISTER.json",
      logical_path: "warehouse-knowledge-base/RULE_REGISTER.json",
      title: "RULE_REGISTER",
      expected_scope: "shared",
    },
  ],

  // yagami-knowledge-base (5 documents)
  [
    "01a1900e-ab2c-7afe-81a8-c264fef62ba2",
    {
      doc_id: "01a1900e-ab2c-7afe-81a8-c264fef62ba2",
      root_key: "yagami-knowledge-base",
      relative_path: "KNOWLEDGE_MANIFEST.json",
      logical_path: "yagami-knowledge-base/KNOWLEDGE_MANIFEST.json",
      title: "KNOWLEDGE_MANIFEST",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab2f-7aad-92d5-8b3fd05d4ec9",
    {
      doc_id: "01a1900e-ab2f-7aad-92d5-8b3fd05d4ec9",
      root_key: "yagami-knowledge-base",
      relative_path: "README.md",
      logical_path: "yagami-knowledge-base/README.md",
      title: "README",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab31-7c16-9edf-79c0e2465d0b",
    {
      doc_id: "01a1900e-ab31-7c16-9edf-79c0e2465d0b",
      root_key: "yagami-knowledge-base",
      relative_path: "yagami-branch-warehouse-map.json",
      logical_path: "yagami-knowledge-base/yagami-branch-warehouse-map.json",
      title: "yagami-branch-warehouse-map",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab33-76f7-8f10-2d8d6de82649",
    {
      doc_id: "01a1900e-ab33-76f7-8f10-2d8d6de82649",
      root_key: "yagami-knowledge-base",
      relative_path: "yagami-branch-warehouse-map.md",
      logical_path: "yagami-knowledge-base/yagami-branch-warehouse-map.md",
      title: "yagami-branch-warehouse-map",
      expected_scope: "shared",
    },
  ],
  [
    "01a1900e-ab36-7719-a26f-b7327f69bbea",
    {
      doc_id: "01a1900e-ab36-7719-a26f-b7327f69bbea",
      root_key: "yagami-knowledge-base",
      relative_path: "yagami-branches.csv",
      logical_path: "yagami-knowledge-base/yagami-branches.csv",
      title: "yagami-branches",
      expected_scope: "shared",
    },
  ],
]);
