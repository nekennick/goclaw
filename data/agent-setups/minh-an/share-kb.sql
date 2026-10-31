-- NOT EXECUTED: pending explicit approval to share with ALL agents in this tenant.
-- Only the 17 snapshotted KB document registrations; no filesystem content writes.
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '20s';
DO $minh_an_share$
DECLARE n integer;
BEGIN
    PERFORM id FROM vault_documents
    WHERE tenant_id = '0193a5b0-7000-7000-8000-000000000001'::uuid
      AND id IN ('01a1900e-ab0b-749c-9808-af69babace6c'::uuid,
        '01a1900e-ab10-7017-ba03-48b32b9774f0'::uuid,
        '01a1900e-ab2f-7aad-92d5-8b3fd05d4ec9'::uuid,
        '01a1900e-ab1f-7bdd-9fcf-9ba994b0b6fb'::uuid,
        '01a1900e-ab23-76d4-876d-ce77634bf019'::uuid,
        '01a1900e-ab17-7e8d-b424-cceb86a42fc5'::uuid,
        '01a1900e-ab1a-70b5-98ee-306d0997bc3a'::uuid,
        '01a1900e-ab1b-7cbe-8647-36b6ac65857a'::uuid,
        '01a1900e-ab26-7800-a3cf-873511ba7f41'::uuid,
        '01a1900e-ab29-7731-83fc-f23a184d7860'::uuid,
        '01a1900e-ab2c-7afe-81a8-c264fef62ba2'::uuid,
        '01a1900e-ab15-7ce3-9233-ee6bc04f9ead'::uuid,
        '01a1900e-ab36-7719-a26f-b7327f69bbea'::uuid,
        '01a1900e-ab11-7b55-946c-78a0a76e9815'::uuid,
        '01a1900e-ab0d-7aee-90cc-c66d35860036'::uuid,
        '01a1900e-ab31-7c16-9edf-79c0e2465d0b'::uuid,
        '01a1900e-ab33-76f7-8f10-2d8d6de82649'::uuid)
    FOR UPDATE;
    SELECT count(*) INTO n FROM vault_documents
    WHERE tenant_id = '0193a5b0-7000-7000-8000-000000000001'::uuid
      AND id IN ('01a1900e-ab0b-749c-9808-af69babace6c'::uuid,
        '01a1900e-ab10-7017-ba03-48b32b9774f0'::uuid,
        '01a1900e-ab2f-7aad-92d5-8b3fd05d4ec9'::uuid,
        '01a1900e-ab1f-7bdd-9fcf-9ba994b0b6fb'::uuid,
        '01a1900e-ab23-76d4-876d-ce77634bf019'::uuid,
        '01a1900e-ab17-7e8d-b424-cceb86a42fc5'::uuid,
        '01a1900e-ab1a-70b5-98ee-306d0997bc3a'::uuid,
        '01a1900e-ab1b-7cbe-8647-36b6ac65857a'::uuid,
        '01a1900e-ab26-7800-a3cf-873511ba7f41'::uuid,
        '01a1900e-ab29-7731-83fc-f23a184d7860'::uuid,
        '01a1900e-ab2c-7afe-81a8-c264fef62ba2'::uuid,
        '01a1900e-ab15-7ce3-9233-ee6bc04f9ead'::uuid,
        '01a1900e-ab36-7719-a26f-b7327f69bbea'::uuid,
        '01a1900e-ab11-7b55-946c-78a0a76e9815'::uuid,
        '01a1900e-ab0d-7aee-90cc-c66d35860036'::uuid,
        '01a1900e-ab31-7c16-9edf-79c0e2465d0b'::uuid,
        '01a1900e-ab33-76f7-8f10-2d8d6de82649'::uuid)
      AND scope = 'personal' AND agent_id = '019ec947-8f55-70fe-8a81-11975e921399'::uuid
      AND team_id IS NULL AND chat_id IS NULL;
    IF n <> 17 THEN
        RAISE EXCEPTION 'Expected exactly 17 unchanged personal KB registrations; got %', n;
    END IF;
    UPDATE vault_documents SET scope='shared', agent_id=NULL, updated_at=NOW()
    WHERE tenant_id = '0193a5b0-7000-7000-8000-000000000001'::uuid
      AND id IN ('01a1900e-ab0b-749c-9808-af69babace6c'::uuid,
        '01a1900e-ab10-7017-ba03-48b32b9774f0'::uuid,
        '01a1900e-ab2f-7aad-92d5-8b3fd05d4ec9'::uuid,
        '01a1900e-ab1f-7bdd-9fcf-9ba994b0b6fb'::uuid,
        '01a1900e-ab23-76d4-876d-ce77634bf019'::uuid,
        '01a1900e-ab17-7e8d-b424-cceb86a42fc5'::uuid,
        '01a1900e-ab1a-70b5-98ee-306d0997bc3a'::uuid,
        '01a1900e-ab1b-7cbe-8647-36b6ac65857a'::uuid,
        '01a1900e-ab26-7800-a3cf-873511ba7f41'::uuid,
        '01a1900e-ab29-7731-83fc-f23a184d7860'::uuid,
        '01a1900e-ab2c-7afe-81a8-c264fef62ba2'::uuid,
        '01a1900e-ab15-7ce3-9233-ee6bc04f9ead'::uuid,
        '01a1900e-ab36-7719-a26f-b7327f69bbea'::uuid,
        '01a1900e-ab11-7b55-946c-78a0a76e9815'::uuid,
        '01a1900e-ab0d-7aee-90cc-c66d35860036'::uuid,
        '01a1900e-ab31-7c16-9edf-79c0e2465d0b'::uuid,
        '01a1900e-ab33-76f7-8f10-2d8d6de82649'::uuid);
    GET DIAGNOSTICS n = ROW_COUNT;
    IF n <> 17 THEN RAISE EXCEPTION 'Unexpected share row count %', n; END IF;
END
$minh_an_share$;
COMMIT;
