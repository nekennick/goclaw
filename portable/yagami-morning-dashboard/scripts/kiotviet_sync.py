#!/usr/bin/env python3
import argparse, csv, datetime as dt, gzip, json, time, urllib.parse, urllib.request, urllib.error
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
ENV_PATH=ROOT/'config/.env.kiotviet'
DATA_ROOT=ROOT/'data/kiotviet'

ENDPOINTS={
    'branches': {'path':'/branches','items_key':'data','params':{'pageSize':100}},
    'customers': {'path':'/customers','items_key':'data','params':{'pageSize':100}},
    'products': {'path':'/products','items_key':'data','params':{'pageSize':100,'includeInventory':'true'}},
    'inventories': {'path':'/inventories','items_key':'data','params':{'pageSize':100}},
    'invoices': {'path':'/invoices','items_key':'data','params':{'pageSize':100}},
}

def now_utc(): return dt.datetime.now(dt.timezone.utc)
def iso_now(): return now_utc().isoformat(timespec='seconds')
def ymd(): return now_utc().strftime('%Y-%m-%d')
def stamp(): return now_utc().strftime('%Y%m%dT%H%M%SZ')

def load_env(path=ENV_PATH):
    env={}
    if not path.exists(): raise SystemExit(f'Missing env file: {path}')
    for line in path.read_text(encoding='utf-8').splitlines():
        line=line.strip()
        if not line or line.startswith('#') or '=' not in line: continue
        k,v=line.split('=',1)
        env[k.strip()]=v.strip().strip('"').strip("'")
    return env

def log_jsonl(path, rec):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('a',encoding='utf-8') as f:
        f.write(json.dumps(rec, ensure_ascii=False)+"\n")

def http_json(req, timeout=60):
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, json.loads(r.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        body=e.read().decode('utf-8', errors='replace')
        raise RuntimeError(f'HTTP {e.code}: {body[:1200]}')

def get_token(env):
    auth=env.get('KIOTVIET_AUTH_URL') or 'https://id.kiotviet.vn/connect/token'
    scope=env.get('KIOTVIET_SCOPE') or 'PublicApi.Access'
    data=urllib.parse.urlencode({
        'scopes': scope,
        'grant_type': 'client_credentials',
        'client_id': env.get('KIOTVIET_CLIENT_ID',''),
        'client_secret': env.get('KIOTVIET_CLIENT_SECRET',''),
    }).encode()
    req=urllib.request.Request(auth, data=data, headers={'Content-Type':'application/x-www-form-urlencoded'}, method='POST')
    status,obj=http_json(req, timeout=30)
    token=obj.get('access_token')
    if not token: raise RuntimeError('No access_token returned')
    return token, obj

def api_get(env, token, path, params):
    base=(env.get('KIOTVIET_API_URL') or 'https://public.kiotapi.com').rstrip('/')
    url=base+path
    if params: url += '?' + urllib.parse.urlencode(params)
    req=urllib.request.Request(url, headers={'Authorization':'Bearer '+token,'Retailer':env['KIOTVIET_RETAILER']}, method='GET')
    return http_json(req, timeout=90)

def fetch_paged(env, token, name, cfg, max_pages=None):
    params=dict(cfg.get('params',{}))
    page_size=int(params.get('pageSize',100))
    current=0; all_items=[]; total=None; page=0
    while True:
        q=dict(params); q['currentItem']=current
        status,obj=api_get(env, token, cfg['path'], q)
        items=obj.get(cfg.get('items_key','data'), []) if isinstance(obj,dict) else []
        if total is None and isinstance(obj,dict): total=obj.get('total')
        if not isinstance(items,list): items=[]
        all_items.extend(items); page+=1
        if not items: break
        current += len(items)
        if total is not None and current >= int(total): break
        if len(items) < page_size: break
        if max_pages and page >= max_pages: break
        time.sleep(0.15)
    return all_items, total

def save_raw(name, items, sync_id, synced_at):
    raw_dir=DATA_ROOT/'raw'/ymd(); raw_dir.mkdir(parents=True, exist_ok=True)
    path=raw_dir/f'{name}-{sync_id}.jsonl.gz'
    with gzip.open(path,'wt',encoding='utf-8') as f:
        for it in items:
            f.write(json.dumps({'source':'KiotViet','endpoint':name,'syncId':sync_id,'syncedAt':synced_at,'data':it}, ensure_ascii=False)+"\n")
    return path

def flatten(x, prefix=''):
    out={}
    if isinstance(x,dict):
        for k,v in x.items():
            nk=f'{prefix}.{k}' if prefix else str(k)
            if isinstance(v,(str,int,float,bool)) or v is None: out[nk]=v
            elif isinstance(v,list): out[nk]=json.dumps(v,ensure_ascii=False)
            elif isinstance(v,dict): out.update(flatten(v,nk))
            else: out[nk]=str(v)
    return out

def save_csv(name, items, synced_at):
    out_dir=DATA_ROOT/'normalized'; out_dir.mkdir(parents=True, exist_ok=True)
    path=out_dir/f'{name}.csv'
    rows=[]; fields=[]; seen=set()
    for it in items:
        r=flatten(it); r['source']='KiotViet'; r['syncedAt']=synced_at
        rows.append(r)
        for k in r.keys():
            if k not in seen: seen.add(k); fields.append(k)
    with path.open('w',encoding='utf-8',newline='') as f:
        w=csv.DictWriter(f, fieldnames=fields, extrasaction='ignore')
        w.writeheader(); w.writerows(rows)
    return path, len(rows)

def write_latest(meta):
    p=DATA_ROOT/'metadata/latest-sync.json'; p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(meta, ensure_ascii=False, indent=2)+"\n", encoding='utf-8')

def main():
    ap=argparse.ArgumentParser(); ap.add_argument('--entities', default='branches,customers,products,inventories'); ap.add_argument('--max-pages', type=int)
    args=ap.parse_args(); env=load_env(); sync_id=stamp(); synced_at=iso_now()
    run={'syncId':sync_id,'startedAt':synced_at,'entities':{},'errors':[]}
    try:
        token,tok=get_token(env); run['auth']={'ok':True,'expiresIn':tok.get('expires_in'),'tokenType':tok.get('token_type')}
    except Exception as e:
        run['auth']={'ok':False,'error':str(e)}; log_jsonl(DATA_ROOT/'logs/sync-log.jsonl', run); raise
    for name in [x.strip() for x in args.entities.split(',') if x.strip()]:
        if name not in ENDPOINTS: run['errors'].append({'entity':name,'error':'unknown entity'}); continue
        start=time.time()
        try:
            items,total=fetch_paged(env, token, name, ENDPOINTS[name], args.max_pages)
            raw=save_raw(name, items, sync_id, synced_at); csvp,count=save_csv(name, items, synced_at)
            run['entities'][name]={'ok':True,'count':count,'apiTotal':total,'raw':str(raw.relative_to(ROOT)),'normalized':str(csvp.relative_to(ROOT)),'seconds':round(time.time()-start,2)}
        except Exception as e:
            err={'entity':name,'error':str(e)}; run['errors'].append(err); log_jsonl(DATA_ROOT/'logs/api-errors.jsonl', {'syncId':sync_id,'syncedAt':synced_at,**err})
    run['finishedAt']=iso_now(); write_latest(run); log_jsonl(DATA_ROOT/'logs/sync-log.jsonl', run)
    print(json.dumps(run, ensure_ascii=False, indent=2))
if __name__=='__main__': main()
