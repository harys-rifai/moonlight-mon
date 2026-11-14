package main
import (
 "fmt"; "log"; "os"; "time"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 now:=time.Now().UTC(); ts:=now.Format("2006-01-02 15:04:05.000Z"); epoch:=now.Unix()
 real:="rf6pna2wr52l3wz"
 exec:=func(q string,p dbx.Params){ if _,err:=db.NewQuery(q).Bind(p).Execute(); err!=nil{log.Fatalf("%v\n%s",err,q)} }
 // dedicated local system
 exec(`delete from system_details where id='seedlocaldt1' or system='seedlocalsys1'`,nil)
 exec(`delete from fingerprints where system='seedlocalsys1'`,nil)
 exec(`delete from network_monitor_stats where monitor in ('seedlocalpg5','seedlocalpgs','seedlocalrds')`,nil)
 exec(`delete from network_monitors where id in ('seedlocalpg5','seedlocalpgs','seedlocalrds')`,nil)
 exec(`delete from systems where id='seedlocalsys1'`,nil)

	exec(`insert into systems (id,name,status,host,port,info,users,created,updated) values ({:id},{:name},'up',{:host},'45876',{:info}::jsonb,{:users}::jsonb,{:created}::timestamp,{:updated}::timestamp)`,
  dbx.Params{"id":"seedlocalsys1","name":"local-machine","host":"localhost","info":`{"h":"local-machine","k":"windows","c":8,"t":16,"m":"Local workstation","u":86400,"cpu":12.5,"mp":45.0,"dp":55.0,"os":2,"v":"0.12.0"}`,"users":fmt.Sprintf("[\"%s\"]",real),"created":ts,"updated":ts})
 exec(`insert into system_details (id,system,hostname,kernel,cores,threads,cpu,os,os_name,arch,memory,podman,updated,sockets) values ({:id},{:system},{:hostname},'Windows',8,16,'Local CPU',2,'Windows 11','x86_64',34359738368,false,{:updated}::timestamp,1)`,
  dbx.Params{"id":"seedlocaldt1","system":"seedlocalsys1","hostname":"local-machine","updated":ts})

 mons:=[]dbx.Params{
  {"id":"seedlocalpg5","system":"seedlocalsys1","target":"localhost","protocol":"tcp","port":5006,"interval":30,"enabled":true,"res":1.8,"avg":1.9,"min":1.2,"max":3.4,"loss":0.0,"created":ts,"updated":ts,"server":"Local PostgreSQL (server db, port 5006)"},
  {"id":"seedlocalpgs","system":"seedlocalsys1","target":"localhost","protocol":"tcp","port":5432,"interval":30,"enabled":true,"res":1.6,"avg":1.7,"min":1.1,"max":3.0,"loss":0.0,"created":ts,"updated":ts,"server":"Local PostgreSQL (default 5432)"},
  {"id":"seedlocalrds","system":"seedlocalsys1","target":"localhost","protocol":"tcp","port":6379,"interval":30,"enabled":true,"res":0.9,"avg":1.0,"min":0.6,"max":2.1,"loss":0.0,"created":ts,"updated":ts,"server":"Local Redis"},
 }
 for _,p:=range mons{
  exec(`insert into network_monitors (id,system,target,protocol,port,interval,enabled,res,"resAvg1h","resMin1h","resMax1h","loss1h",created,updated,"certInfo",server,"skipTlsVerify") values ({:id},{:system},{:target},{:protocol},{:port},{:interval},{:enabled},{:res},{:avg},{:min},{:max},{:loss},{:created}::timestamp,{:updated}::timestamp,null,{:server},false)`, p)
  exec(`insert into network_monitor_stats (id,system,monitor,total_count,success_count,res_sum,res_min,res_max,type,created) values ({:id},{:system},{:monitor},120,120,{:sum},{:rmin},{:rmax},'1m',{:created})`,
   dbx.Params{"id":"seedstat"+p["id"].(string)[8:],"system":"seedlocalsys1","monitor":p["id"],"sum":200000,"rmin":800,"rmax":3400,"created":epoch})
 }
 fmt.Println("added local system + postgres/redis monitors")
}
