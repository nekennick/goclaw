import json,csv,gzip,time,urllib.parse,urllib.request,urllib.error,datetime as dt,unicodedata
from pathlib import Path
ROOT=Path('.').resolve(); DATA=ROOT/'data/kiotviet'; TZ=dt.timezone(dt.timedelta(hours=7))
def env():
 d={}
 for l in (ROOT/'config/.env.kiotviet').read_text().splitlines():
  l=l.strip()
  if l and not l.startswith('#') and '=' in l:
   k,v=l.split('=',1); d[k.strip()]=v.strip().strip('"').strip("'")
 return d
def hj(req,to=90):
 with urllib.request.urlopen(req,timeout=to) as r: return json.loads(r.read().decode())
def tok(e):
 data=urllib.parse.urlencode({'scopes':e.get('KIOTVIET_SCOPE','PublicApi.Access'),'grant_type':'client_credentials','client_id':e['KIOTVIET_CLIENT_ID'],'client_secret':e['KIOTVIET_CLIENT_SECRET']}).encode()
 return hj(urllib.request.Request(e.get('KIOTVIET_AUTH_URL','https://id.kiotviet.vn/connect/token'),data=data,headers={'Content-Type':'application/x-www-form-urlencoded'},method='POST'),30)['access_token']
def api(e,t,path,params):
 url=e.get('KIOTVIET_API_URL','https://public.kiotapi.com').rstrip('/')+path+'?'+urllib.parse.urlencode(params)
 return hj(urllib.request.Request(url,headers={'Authorization':'Bearer '+t,'Retailer':e['KIOTVIET_RETAILER']}))
def all_(e,t,path,params,maxp=None):
 out=[]; cur=0; total=None; ps=int(params.get('pageSize',100)); p=0
 while 1:
  q=dict(params); q['currentItem']=cur; o=api(e,t,path,q); arr=o.get('data',[]); total=o.get('total') if total is None else total
  if not arr: break
  out+=arr; cur+=len(arr); p+=1
  if (total and cur>=total) or len(arr)<ps or (maxp and p>=maxp): break
  time.sleep(.1)
 return out,total
def flat(x):
 r={}
 for k,v in x.items(): r[k]=json.dumps(v,ensure_ascii=False) if isinstance(v,(list,dict)) else v
 return r
def wc(path,rows):
 path.parent.mkdir(parents=True,exist_ok=True); fs=[]
 for r in rows:
  for k in r:
   if k not in fs: fs.append(k)
 with path.open('w',encoding='utf-8',newline='') as f:
  w=csv.DictWriter(f,fieldnames=fs,extrasaction='ignore'); w.writeheader(); w.writerows(rows)
def raw(name,items,stamp,synced):
 d=DATA/'raw'/dt.datetime.now(TZ).strftime('%Y-%m-%d'); d.mkdir(parents=True,exist_ok=True)
 with gzip.open(d/f'{name}-{stamp}.jsonl.gz','wt',encoding='utf-8') as f:
  for it in items: f.write(json.dumps({'source':'KiotViet','endpoint':name,'syncId':stamp,'syncedAt':synced,'data':it},ensure_ascii=False)+'\n')
def plocal(s):
 if not s: return None
 return dt.datetime.fromisoformat(s[:19])
def money(x): return f"{float(x or 0):,.0f}".replace(',','.')
def main():
 now=dt.datetime.now(TZ); stamp=dt.datetime.utcnow().strftime('%Y%m%dT%H%M%SZ'); synced=dt.datetime.utcnow().replace(microsecond=0,tzinfo=dt.timezone.utc).isoformat(); e=env(); t=tok(e)
 meta={'syncId':stamp,'startedAt':synced,'entities':{}}
 products=[]
 for n,p,pa in [('branches','/branches',{'pageSize':100}),('customers','/customers',{'pageSize':100}),('products','/products',{'pageSize':100,'includeInventory':'true'})]:
  its,total=all_(e,t,p,pa); raw(n,its,stamp,synced); wc(DATA/f'normalized/{n}.csv',[flat(x) for x in its]); meta['entities'][n]={'ok':True,'count':len(its),'apiTotal':total}
  if n=='products': products=its
 inv=[]
 for p in products:
  for i in p.get('inventories') or []:
   r={k:i.get(k) for k in ['productId','productCode','productName','branchId','branchName','cost','onHand','reserved','onOrder','isActive']}; r['available']=(r.get('onHand') or 0)-(r.get('reserved') or 0); r['syncedAt']=synced; inv.append(r)
 wc(DATA/'normalized/inventory-current.csv',inv); (DATA/'metadata/latest-sync.json').write_text(json.dumps(meta,ensure_ascii=False,indent=2),encoding='utf-8')
 start=(now-dt.timedelta(days=now.weekday())).replace(hour=0,minute=0,second=0,microsecond=0); end=now
 orders,totalo=all_(e,t,'/orders',{'pageSize':100,'orderBy':'purchaseDate','orderDirection':'Desc'},8); orders=[o for o in orders if plocal(o.get('purchaseDate')) and start.replace(tzinfo=None)<=plocal(o.get('purchaseDate'))<=end.replace(tzinfo=None)]
 invoices,totali=all_(e,t,'/invoices',{'pageSize':100,'fromPurchaseDate':start.strftime('%Y-%m-%dT%H:%M:%S'),'toPurchaseDate':end.strftime('%Y-%m-%dT%H:%M:%S')})
 raw('orders-week-current',orders,stamp,synced); raw('invoices-week-current',invoices,stamp,synced); wc(DATA/'normalized/orders-week-current.csv',[flat(x) for x in orders]); wc(DATA/'normalized/invoices-week-current.csv',[flat(x) for x in invoices])
 kb=json.load(open(ROOT/'data/yagami-knowledge-base/yagami-branch-warehouse-map.json',encoding='utf-8')); day=['THỨ 2','THỨ 3','THỨ 4','THỨ 5','THỨ 6','THỨ 7','CHỦ NHẬT'][now.weekday()]
 sched=[b for b in kb['branches'] if b.get('status')=='ACTIVE' and b.get('day')==day and b.get('customerCode')]; sc={b['customerCode']:b for b in sched}
 os=[o for o in orders if o.get('customerCode') in sc]
 byc={}
 for o in os: byc.setdefault(o.get('customerCode'),[]).append(o)
 invok={}
 for i in invoices:
  if i.get('orderCode') and i.get('statusValue')=='Hoàn thành': invok.setdefault(i['orderCode'],[]).append(i)
 done=[]; pend=[]; reco=[]
 for o in os:
  r={'branch':o.get('customerName'),'code':o.get('code'),'warehouse':o.get('branchName'),'total':money(o.get('total')),'status':o.get('statusValue')}
  has=o.get('code') in invok; multi=len(byc.get(o.get('customerCode'),[]))>1
  if has and not multi: done.append(r)
  elif o.get('statusValue')=='Phiếu tạm' and not has and not multi: pend.append(r)
  else: reco.append(r)
 miss=[b for b in sched if b['customerCode'] not in byc]
 codes=['myg','sml','ca','xxm']; inv4=[r for r in inv if str(r.get('productCode','')).lower() in codes and r.get('branchName') in ['Tổng kho YAGAMI','KHO YAGAMI BÌNH DƯƠNG']]
 res={'now':now.strftime('%d/%m/%Y %H:%M:%S'),'syncLocal':dt.datetime.fromisoformat(synced).astimezone(TZ).strftime('%d/%m/%Y %H:%M:%S'),'day':day,'week':start.strftime('%d/%m')+'-'+end.strftime('%d/%m %H:%M'),'sched':len(sched),'withOrders':len(byc),'missing':len(miss),'totalOrders':len(os),'done':done,'pending':pend,'reconcile':reco,'missingList':miss,'inv4':inv4,'kbv':kb['version'],'kbdate':kb['verifiedAt']}
 Path('outputs').mkdir(exist_ok=True); Path('outputs/afternoon-report.json').write_text(json.dumps(res,ensure_ascii=False,indent=2),encoding='utf-8'); print(json.dumps(res,ensure_ascii=False))
main()
